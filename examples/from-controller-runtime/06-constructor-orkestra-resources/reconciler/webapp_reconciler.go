//go:build ignore

// reconciler/webapp_reconciler.go
//
// The WebApp constructor using Orkestra's pkg/resources library.
//
// Compare with 04-constructor-migration/reconciler/webapp_reconciler.go.
// The signature, struct, and constructor function are identical. Only
// reconcileDeployment and reconcileService change — the Get / IsNotFound /
// Create / Patch pattern collapses into a single Update call.
//
// Before (manual — from 04-constructor-migration):
//
//	existing := &appsv1.Deployment{}
//	err := r.kube.Get(ctx, ns, name, existing)
//	if errors.IsNotFound(err) { return r.kube.Create(ctx, desired) }
//	patch := sigs.MergeFrom(existing.DeepCopy())
//	existing.Spec = desired.Spec
//	return r.kube.Patch(ctx, existing, patch)
//
// After (Orkestra resources):
//
//	return orkdeploy.Update(ctx, r.kube, webapp, spec)
//
// Update handles: create-if-absent, drift correction, owner references,
// system labels (managed-by: orkestra, orkestra-owner: <cr-name>).
// DeleteIfOwned removes a resource only if this CR owns it — no-op otherwise.
package reconciler

import (
	"context"
	"fmt"

	apiv1 "github.com/orkspace/from-controller-runtime-demo/api/v1alpha1"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orkdeploy "github.com/orkspace/orkestra/pkg/resources/deployments"
	orksvc "github.com/orkspace/orkestra/pkg/resources/services"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebAppReconciler implements domain.Reconciler for the WebApp CRD.
type WebAppReconciler struct {
	kube kubeclient.Interface
}

// NewWebAppReconciler is the constructor function registered in the Katalog.
func NewWebAppReconciler(kube kubeclient.Interface) domain.Reconciler {
	return &WebAppReconciler{kube: kube}
}

// Reconcile is called by Orkestra's worker pool for every queued WebApp key.
func (r *WebAppReconciler) Reconcile(ctx context.Context, req domain.Request) (domain.Result, error) {
	if req.Prepared == nil {
		return domain.Result{}, nil
	}

	webapp, err := domain.ToTyped[apiv1.WebApp](req.Prepared)
	if err != nil {
		return domain.Result{}, fmt.Errorf("toTyped: %w", err)
	}

	if webapp.DeletionTimestamp != nil {
		return domain.Result{}, nil
	}

	if err := r.reconcileDeployment(ctx, webapp); err != nil {
		return domain.Result{}, err
	}
	if err := r.reconcileService(ctx, webapp); err != nil {
		return domain.Result{}, err
	}

	r.kube.GetEventRecorder().Eventf(webapp, corev1.EventTypeNormal, "WebAppReconciled",
		"WebApp %s/%s reconciled", webapp.Namespace, webapp.Name)

	return domain.Result{}, r.kube.PatchStatus(ctx, webapp, map[string]interface{}{
		"phase":    "Running",
		"endpoint": fmt.Sprintf("%s-svc.%s.svc.cluster.local", webapp.Name, webapp.Namespace),
		"replicas": webapp.Spec.Replicas,
	}, metav1.PatchOptions{})
}

// reconcileDeployment — single Update call replaces Get/IsNotFound/Create/Patch.
// Owner references, system labels, and idempotency are all handled by Update.
func (r *WebAppReconciler) reconcileDeployment(ctx context.Context, webapp *apiv1.WebApp) error {
	spec := orkdeploy.Resolve(
		orktypes.DeploymentTemplateSource{
			Name:      webapp.Name,
			Namespace: webapp.Namespace,
			Image:     webapp.Spec.Image,
			Replicas:  fmt.Sprintf("%d", webapp.Spec.Replicas),
			Port:      fmt.Sprintf("%d", webapp.Spec.Port),
		},
		webapp.Name,
		nil,
	)
	if err := orkdeploy.Update(ctx, r.kube, webapp, spec); err != nil {
		return fmt.Errorf("webapp deployment: %w", err)
	}
	return nil
}

// reconcileService — single Update call replaces Get/IsNotFound/Create/Patch.
func (r *WebAppReconciler) reconcileService(ctx context.Context, webapp *apiv1.WebApp) error {
	spec := orksvc.Resolve(
		orktypes.ServiceTemplateSource{
			Name:       webapp.Name + "-svc",
			Namespace:  webapp.Namespace,
			Port:       "80",
			TargetPort: fmt.Sprintf("%d", webapp.Spec.Port),
			Type:       "ClusterIP",
		},
		webapp.Name,
	)
	if err := orksvc.Update(ctx, r.kube, webapp, spec); err != nil {
		return fmt.Errorf("webapp service: %w", err)
	}
	return nil
}
