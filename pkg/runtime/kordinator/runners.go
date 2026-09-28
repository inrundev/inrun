package kordinator

import (
	"context"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/contract"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/maintain"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/prepare"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// runCleanupGate evaluates operatorBox.runtime.cleanup.
// triggered=true  → delete this cycle.
// waitFor>0       → grace period running; re-enqueue after waitFor.
// both zero       → conditions not yet met; reconcile runs normally.
func (k *Kontroller) runCleanupGate(
	ctx context.Context,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
	resolver *orktmpl.Resolver,
) (triggered bool, waitFor time.Duration, err error) {
	return k.evaluateCleanup(ctx, obj, box, resolver)
}

// runMaintain applies labels, annotations, and finalizers before reconcile.
// ForCleanup=true strips deletion-protection so the subsequent Delete call
// bypasses the webhook selector.
func (k *Kontroller) runMaintain(
	ctx context.Context,
	entry contract.RegistryEntry,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
	resolver *orktmpl.Resolver,
	forCleanup bool,
) error {
	return maintain.Apply(ctx, maintain.Input{
		CRD:        entry.CRD,
		Kat:        k.kat,
		Kube:       k.kube,
		Recorder:   k.event,
		ForCleanup: forCleanup,
	}, obj, box, resolver)
}

// resolveRequeueAfter returns the post-reconcile requeue delay.
// 0 means re-enqueue immediately; -1 means wait for an informer event.
func (k *Kontroller) resolveRequeueAfter(
	ctx context.Context,
	gvk string,
	entry contract.RegistryEntry,
	key string,
	result domain.Result,
	prepared *domain.PreparedRequest,
) time.Duration {
	if result.RequeueAfter > 0 {
		return result.RequeueAfter
	}
	obj := k.objectFromCache(entry, key)
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
	if d := k.kat.EvaluateRequeue(ctx, entry.CRD.Name, obj, resolver); d > 0 {
		return d
	}
	if prepared != nil {
		if box := prepare.BoxFrom(prepared); box.HasCleanup() {
			return 0
		}
	}
	return -1
}
