package kordinator

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/maintain"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/post"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/prepare"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/vitals"
	"github.com/orkspace/orkestra/pkg/runtime/queue"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// Worker that only processes items for a specific GVK
func (k *Kontroller) runWorkerForGVK(ctx context.Context, gvk string, workerID string) {
	wq, ok := k.queueRegistry.For(gvk)

	if !ok {
		logger.Warn().Str("gvk", gvk).Msg("no queue for CRD. Using default queue")
		wq = k.defaultWorkqueue
	}

	logger.Debug().Str("worker_id", workerID).Str("gvk", gvk).Msg("worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Debug().Str("worker_id", workerID).Str("gvk", gvk).Msg("worker stopped")
			return
		default:
			// Worker is idle - waiting for work
			// (already idle from previous iteration or initialization)

			// Wait for an item
			item, shutdown := wq.Get()
			if shutdown {
				logger.Debug().Str("worker_id", workerID).Str("gvk", gvk).Msg("worker stopping (queue shutdown)")
				return
			}

			// Worker is now processing
			k.crdHealthMap[gvk].MarkWorkerProcessing(workerID)

			// Process the item
			func() {
				defer wq.Done(item)
				k.processItemForGVK(ctx, gvk, item)
			}()

			// Back to idle after processing
			k.crdHealthMap[gvk].MarkWorkerIdle(workerID)

			// Update metrics (outside of processing state)
			depth := float64(wq.Depth())
			metrics.SetQueueDepth(gvk, depth)

			// Push live depth into AutoMetrics so the autoscaler can read it.
			// No-op for non-autoscaled CRDs (runtimeMap entry has no autoscaler).
			k.mu.RLock()
			rt := k.runtimeMap[gvk]
			k.mu.RUnlock()
			if rt != nil {
				rt.autoMetrics.SetQueueDepth(int64(wq.Depth()))
			}

			// Resource count — read from this CRD's informer cache
			if entry, ok := k.katalog.Get(gvk); ok && entry.Informer != nil {
				count := float64(len(entry.Informer.GetIndexer().List()))
				metrics.SetResourceCount(gvk, count)
			}
		}
	}
}

// processItemForGVK handles a single reconciliation item
func (k *Kontroller) processItemForGVK(ctx context.Context, gvk string, item queue.QueueItem) {
	if err := ctx.Err(); err != nil {
		logger.Debug().Msg("process item: context cancelled")
		return
	}

	// Resolve queue — per-CRD if registered, default otherwise
	wq, ok := k.queueRegistry.For(gvk)
	if !ok {
		wq = k.defaultWorkqueue
	}

	// Added to help shutdown workers and preserve the queue on missing crds
	// After dequeuing, they check ctx.Done() at the top of the loop and exit.
	// The queue is intact for reactivation. No ShutDown() called.
	// TODO: Currently does not shutdown the workers
	if item.Key == drainSentinel {
		wq.Forget(item)
		return
	}

	// With per-CRD queues this check is only needed for the default queue path
	// where multiple GVKs share one queue
	if item.GVK != gvk {
		if ok {
			// This item is in the wrong per-CRD queue — should not happen
			// Log and drop rather than spin
			logger.Error().
				Str("expected", gvk).
				Str("got", item.GVK).
				Msg("GVK mismatch in per-CRD queue — dropping item")
			wq.Forget(item)
		} else {
			// Default queue — item belongs to a different GVK, put it back
			// This is the only valid re-queue case
			wq.AddRateLimited(item)
		}
		return
	}

	// Look up the pre-built reconciler
	k.mu.RLock()
	rec := k.reconcilers[gvk]
	k.mu.RUnlock()

	if rec == nil {
		logger.Error().Str("gvk", gvk).Str("key", item.Key).Msg("no reconciler found — dropping item")
		wq.Forget(item)
		return
	}

	// Pre-reconcile gate: evaluate operatorBox.preReconcile.when/or conditions.
	// The reconciler is never called when conditions are not met — gated state
	// is idle, not failure; error rate and health state are unaffected.
	entry, hasEntry := k.katalog.Get(gvk)
	if hasEntry && entry.CRD.HasAnyReconcileGate() {
		obj := k.objectFromCache(entry, item.Key)
		sentinelMap := wq.Sentinels(item)
		if gated, reason := k.evaluatePreReconcileCheck(ctx, obj, entry.CRD.Name, sentinelMap); gated {
			k.crdHealthMap[gvk].RecordGated(reason)
			wq.Forget(item)
			return
		}
	}

	// Prepare: build the enriched request (normalize, resolver, cross, validation…).
	// Returns nil when the object is not in cache (deleted between dequeue and here)
	// or when the namespace guard blocks it — both are silent skips, not errors.
	var prepared *domain.PreparedRequest
	var valResult *prepare.ValidationResult
	if hasEntry {
		var prepErr error
		prepared, valResult, prepErr = prepare.Prepare(ctx, prepare.Input{
			Entry:    entry,
			Key:      item.Key,
			Kat:      k.kat,
			Kube:     k.kube,
			Registry: k.katalog,
		})
		if prepErr != nil {
			logger.Error().Err(prepErr).Str("gvk", gvk).Str("key", item.Key).Msg("prepare failed")
			wq.AddRateLimited(item)
			k.failedReconcile(gvk)
			return
		}
		if prepared == nil {
			wq.Forget(item)
			return
		}
	}

	// Maintain: apply labels, annotations, and finalizers before reconcile.
	if hasEntry && prepared != nil {
		box := prepare.BoxFrom(prepared)
		resolver := prepared.Context.(*orktmpl.Resolver)
		if maintErr := maintain.Apply(ctx, maintain.Input{
			CRD:      entry.CRD,
			Kat:      k.kat,
			Kube:     k.kube,
			Recorder: k.event,
		}, prepared.Object, box, resolver); maintErr != nil {
			logger.Error().Err(maintErr).Str("gvk", gvk).Str("key", item.Key).Msg("maintain failed")
			wq.AddRateLimited(item)
			k.failedReconcile(gvk)
			return
		}
	}

	// safeReconcile catches panics
	result, reconcileErr := k.safeReconcile(rec, k.crdHealthMap[gvk], ctx, item.Key, gvk, prepared)

	// Post: status patch + emit — always runs, even on reconcile failure.
	if hasEntry && prepared != nil {
		box := prepare.BoxFrom(prepared)
		resolver := prepared.Context.(*orktmpl.Resolver)
		health := k.crdHealthMap[gvk]
		post.Apply(ctx, post.Input{
			CRD:        entry.CRD,
			Kube:       k.kube,
			Recorder:   k.event,
			MetricsMap: health.GetAutoMetrics(),
			HealthMap:  health.HealthAsMap(),
		}, prepared.Object, resolver, box, reconcileErr, toPostValResult(valResult))
	}

	if reconcileErr != nil {
		logger.Error().Err(reconcileErr).Str("gvk", gvk).Str("key", item.Key).Msg("reconcile failed")
		wq.AddRateLimited(item)
		k.failedReconcile(gvk)
		return
	}

	wq.Forget(item)

	requeueAfter := result.RequeueAfter
	if requeueAfter == 0 {
		if hasEntry {
			obj := k.objectFromCache(entry, item.Key)
			var resolver *orktmpl.Resolver
			if prepared != nil {
				resolver = prepared.Context.(*orktmpl.Resolver)
			} else if obj != nil {
				if r, err := orktmpl.NewResolver(ctx, obj); err == nil {
					health := k.crdHealthMap[gvk]
					resolver = r.WithUserNotes(k.kat.UserNotes()).
						WithProfiles(k.kat.UserProfiles()).
						WithHealth(health.HealthAsMap()).
						WithMetrics(health.GetAutoMetrics())
				}
			}
			requeueAfter = k.kat.EvaluateRequeue(ctx, entry.CRD.Name, obj, resolver)
		}
	}
	if requeueAfter > 0 {
		wq.AddAfter(item, requeueAfter)
	}
}

// safeReconcile wraps a Reconciler's Reconcile() call in a fully isolated,
// panic‑protected execution boundary. This is the core of Orkestra’s
// "operator sandbox" model: each CRD runs in its own safe compartment,
// ensuring that failures in one operator never cascade into others.
//
// Responsibilities:
//   - Measure reconcile duration for metrics
//   - Catch and convert panics into errors (preventing controller crash)
//   - Record success/failure into CRDHealth
//   - Emit success/error metrics
//   - Apply degrade thresholds for health tracking
//
// This function guarantees that no matter what happens inside rec.Reconcile(),
// the controller process stays alive and the failure is reported deterministically.
func (k *Kontroller) safeReconcile(
	rec domain.Reconciler,
	health *vitals.CRDHealth,
	ctx context.Context,
	key string,
	gvk string,
	prepared *domain.PreparedRequest,
) (result domain.Result, err error) {

	// Track how long this reconcile took.
	// The defer ensures duration is recorded even if a panic occurs.
	start := time.Now()
	defer func() {
		metrics.ObserveReconcileDuration(gvk, time.Since(start).Seconds())

		// Panic recovery: this is the isolation boundary.
		// Any panic inside the operator is caught, logged, and converted into an error.
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)

			err = fmt.Errorf("reconciler panic: %v", r)

			logger.Error().
				Str("gvk", gvk).
				Str("key", key).
				Str("panic", fmt.Sprint(r)).
				Str("stack", string(buf[:n])).
				Msg("reconciler panic recovered")

			// Update CRD health state and metrics.
			health.RecordFailure(err, k.failureThreshold[gvk])
			metrics.RecordReconcile(gvk, "error")

			// TODO: track panic stats differently with recovery
		}
	}()

	// Inject per-request fields into the logr context so ctrl.LoggerFrom(ctx)
	// automatically carries the resource key on every reconciler log line.
	ctx = ctrllog.IntoContext(ctx, ctrllog.FromContext(ctx).WithValues("resource", key, "gvk", gvk))

	// Execute the operator's reconcile logic.
	// Any returned error is treated as a reconcile failure.
	ns, name, _ := cache.SplitMetaNamespaceKey(key)
	result, err = rec.Reconcile(ctx, domain.Request{
		Key:            key,
		NamespacedName: apitypes.NamespacedName{Namespace: ns, Name: name},
		Prepared:       prepared,
	})
	if err != nil {
		// Update CRD health state and metrics.
		health.RecordFailure(err, k.failureThreshold[gvk])
		metrics.RecordReconcile(gvk, "error")
		return result, err
	}

	// Successful reconcile path.
	health.RecordSuccess()
	k.successReconcile(gvk)
	metrics.RecordReconcile(gvk, "success")
	return result, nil
}

// toPostValResult converts a prepare.ValidationResult to the post package's
// equivalent type for status condition writing.
func toPostValResult(v *prepare.ValidationResult) *post.ValidationResult {
	if v == nil {
		return nil
	}
	r := &post.ValidationResult{Deny: v.Deny}
	for _, viol := range v.Violations {
		r.Violations = append(r.Violations, post.ValidationViolation{
			Field:   viol.Field,
			Rule:    viol.Rule,
			Value:   viol.Value,
			Message: viol.Message,
		})
	}
	for _, w := range v.Warnings {
		r.Warnings = append(r.Warnings, post.ValidationViolation{
			Field:   w.Field,
			Rule:    w.Rule,
			Value:   w.Value,
			Message: w.Message,
		})
	}
	return r
}
