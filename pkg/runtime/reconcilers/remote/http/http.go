package http

import (
	"context"
	"fmt"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const defaultRemoteTimeout = 30 * time.Second

// Reconciler implements domain.Reconciler by POSTing a PreparedRequest
// to a remote HTTP endpoint. The remote service owns the reconcile logic;
// Orkestra owns the queue, backoff, informer, health, and metrics.
type Reconciler struct {
	decl             *orktypes.RemoteReconcilerDeclaration
	gvk              schema.GroupVersionKind
	kube             kubeclient.Interface
	event            event.Recorder
	managedResources []domain.ManagedResource
	namespace        string // orkestra namespace, used as default when secretRef.namespace is empty
	client           httpDoer
}

// New constructs a Reconciler for the given CRD.
// kube is used for credential resolution and SSA-applying resources returned by the remote reconciler.
// ev is used to emit Kubernetes events on reconcile success or failure.
// managedResources is the set of types the remote reconciler is permitted to create.
// ownNamespace is the Orkestra namespace (used when secretRef.namespace is empty).
func New(
	decl *orktypes.RemoteReconcilerDeclaration,
	gvk schema.GroupVersionKind,
	kube kubeclient.Interface,
	ev event.Recorder,
	managedResources []domain.ManagedResource,
	ownNamespace string,
) *Reconciler {
	timeout := defaultRemoteTimeout
	if decl.Timeout.Duration > 0 {
		timeout = decl.Timeout.Duration
	}
	return &Reconciler{
		decl:             decl,
		gvk:              gvk,
		kube:             kube,
		event:            ev,
		managedResources: managedResources,
		namespace:        ownNamespace,
		client:           newHTTPClient(timeout),
	}
}

// Reconcile dispatches the request to the remote endpoint, resolves the result,
// applies returned resources, and emits a Kubernetes event for the outcome.
func (r *Reconciler) Reconcile(ctx context.Context, req domain.Request) (domain.Result, error) {
	log := logger.FromContext(ctx).With().
		Str("reconciler", "remote").
		Str("endpoint", r.resolveEndpoint(req)).
		Logger()

	log.Info().Str("key", req.Key).Msg("dispatching to remote reconciler")

	raw, err := r.callRemote(ctx, req)
	if err != nil {
		return domain.Result{}, err
	}

	if err := r.applyAndStore(ctx, req, raw.Resources); err != nil {
		return domain.Result{}, err
	}

	return r.toResult(ctx, req, raw)
}

// applyAndStore resolves, applies and caches the resources returned by the remote
// service, then emits a Kubernetes event for the outcome.
func (r *Reconciler) applyAndStore(ctx context.Context, req domain.Request, rawResources []map[string]interface{}) error {
	log := logger.FromContext(ctx).With().
		Str("reconciler", "remote").
		Logger()

	var owner domain.Object
	if req.Prepared != nil {
		owner = req.Prepared.Object
	}

	resources, err := r.resolveResources(rawResources, req)
	if err != nil {
		return fmt.Errorf("remote reconciler: resolve resources: %w", err)
	}

	if err := r.apply(ctx, owner, resources); err != nil {
		r.event.Eventf(owner, corev1.EventTypeWarning, r.gvk.Kind+"ReconcileError",
			"Remote reconciler failed for %s/%s: %v", owner.GetNamespace(), owner.GetName(), err)
		return err
	}

	r.event.Eventf(owner, corev1.EventTypeNormal, r.gvk.Kind+"Reconciled",
		"Remote reconciler succeeded for %s/%s", owner.GetNamespace(), owner.GetName())
	log.Info().Str("key", req.Key).Msg("remote reconciler returned ok")
	return nil
}
