package kordinator

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/orkspace/orkestra/pkg/logger"
	ork_autoscaler "github.com/orkspace/orkestra/pkg/runtime/autoscaler"
	orkqueue "github.com/orkspace/orkestra/pkg/runtime/queue"
)

// perCRDRuntime holds the per-CRD concurrency and autoscale state owned by
// the Kontroller. Previously embedded in generic.Reconciler.
type perCRDRuntime struct {
	sem         *ork_autoscaler.ResizableSemaphore
	autoMetrics *ork_autoscaler.AutoMetrics
	autoscaler  *ork_autoscaler.Autoscaler
	resyncNs    atomic.Int64
	spawnWorker func()
}

// kordinatorTarget implements autoscaler.AutoscaleTarget backed by the
// Kontroller's owned state for a specific GVK. Created by startCRDWorkers.
type kordinatorTarget struct {
	rt  *perCRDRuntime
	wq  *orkqueue.Workqueue
	gvk string
}

func (t *kordinatorTarget) ResizeWorkers(n int) {
	old := t.rt.sem.Capacity()
	t.rt.sem.Resize(n)
	if n > old && t.rt.spawnWorker != nil {
		for i := old; i < n; i++ {
			go t.rt.spawnWorker()
		}
	}
	logger.Info().
		Str("crd", t.gvk).
		Int("workers", n).
		Msg("autoscaler: worker pool resized")
}

func (t *kordinatorTarget) SetQueueDepthLimit(n int) {
	if t.wq != nil {
		t.wq.SetQueueDepth(n)
	}
	logger.Info().
		Str("crd", t.gvk).
		Int("queueDepth", n).
		Msg("autoscaler: queue depth limit updated")
}

func (t *kordinatorTarget) SetResyncInterval(d time.Duration) {
	t.rt.resyncNs.Store(d.Nanoseconds())
	logger.Info().
		Str("crd", t.gvk).
		Dur("resync", d).
		Msg("autoscaler: resync interval updated")
}

// startResyncLoop runs the adjustable resync goroutine for a CRD.
// Idle (polling 500ms) when resyncNs == 0; fires at the stored interval otherwise.
// Mirrors the resync logic that was previously on generic.Reconciler.
func (k *Kontroller) startResyncLoop(ctx context.Context, gvk string) {
	rt := k.runtimeMap[gvk]
	entry, ok := k.katalog.Get(gvk)
	if !ok || entry.Informer == nil {
		return
	}
	wq, _ := k.queueRegistry.For(gvk)
	go func() {
		for {
			ns := rt.resyncNs.Load()
			if ns == 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(500 * time.Millisecond):
				}
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(ns)):
				if wq == nil {
					continue
				}
				items := entry.Informer.GetIndexer().List()
				for _, obj := range items {
					wq.Enqueue(obj, gvk)
				}
				logger.Debug().
					Str("crd", gvk).
					Int("count", len(items)).
					Dur("interval", time.Duration(ns)).
					Msg("autoscaler: resync re-enqueued all objects")
			}
		}
	}()
}
