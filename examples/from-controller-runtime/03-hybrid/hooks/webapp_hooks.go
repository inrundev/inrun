//go:build ignore

// hooks/webapp_hooks.go
//
// Hybrid WebApp hook — the Service is created here in Go.
// The Deployment is still declared in the Katalog's operatorBox.onCreate.
//
// Orkestra still provides:
//   - Informer watching the WebApp CRD
//   - Workqueue with deduplication and backoff
//   - Worker pool
//   - Finalizer management
//   - Kubernetes events
//   - Status management (Layer 1 Ready condition)
//   - Prometheus metrics
//   - The Deployment (created by the declarative operatorBox)
//
// This hook provides:
//   - Type-safe struct access to WebApp spec fields
//   - Service creation with the exact spec your team controls
package hooks

import (
	"context"
	"fmt"

	apiv1 "github.com/orkspace/from-controller-runtime-demo/api/v1alpha1"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orksvc "github.com/orkspace/orkestra/pkg/resources/services"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// WebAppHooks returns the hook implementation for the WebApp CRD.
func WebAppHooks() domain.AnyReconcileHooks {
	return domain.ReconcileHooks[*apiv1.WebApp]{
		OnReconcile: onWebAppReconcile,
	}
}

// onWebAppReconcile runs on every reconcile cycle.
// Creates the Service — the Deployment is handled by the declarative operatorBox.
func onWebAppReconcile(ctx context.Context, obj *apiv1.WebApp) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not in context")
	}

	svcSpec := orksvc.Resolve(
		orktypes.ServiceTemplateSource{
			Name:       obj.Name + "-svc",
			Namespace:  obj.Namespace,
			Port:       "80",
			TargetPort: fmt.Sprintf("%d", obj.Spec.Port),
			Type:       "ClusterIP",
		},
		obj.Name,
	)
	if err := orksvc.Update(ctx, kube, obj, svcSpec); err != nil {
		return fmt.Errorf("webapp service: %w", err)
	}

	return nil
}
