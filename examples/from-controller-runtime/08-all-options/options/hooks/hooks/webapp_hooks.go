//go:build ignore

// hooks/webapp_hooks.go
//
// Hooks-only HooksApp hook — both resources are created here in Go.
// No declared templates in the Katalog. The hook owns the full resource set.
//
// Compare with 02-hybrid/hooks/webapp_hooks.go where only the Service is here;
// the Deployment is declared in the Katalog.
//
// Use hooks-only when every resource requires computed values or type-safe
// control that templates cannot express.
package hooks

import (
	"context"
	"fmt"

	apiv1 "github.com/orkspace/from-controller-runtime-all-options/options/hooks/api/v1alpha1"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orkdeploy "github.com/orkspace/orkestra/pkg/resources/deployments"
	orksvc "github.com/orkspace/orkestra/pkg/resources/services"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// WebAppHooks returns the hook implementation for the HooksApp CRD.
func WebAppHooks() domain.AnyReconcileHooks {
	return domain.ReconcileHooks[*apiv1.HooksApp]{
		OnReconcile: onWebAppReconcile,
	}
}

func onWebAppReconcile(ctx context.Context, obj *apiv1.HooksApp) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not in context")
	}

	// ── Deployment ────────────────────────────────────────────────────────
	deploySpec := orkdeploy.Resolve(
		orktypes.DeploymentTemplateSource{
			Name:      obj.Name,
			Namespace: obj.Namespace,
			Image:     obj.Spec.Image,
			Replicas:  fmt.Sprintf("%d", obj.Spec.Replicas),
			Port:      fmt.Sprintf("%d", obj.Spec.Port),
		},
		obj.Name,
		nil,
	)
	if err := orkdeploy.Update(ctx, kube, obj, deploySpec); err != nil {
		return fmt.Errorf("webapp deployment: %w", err)
	}

	// ── Service ───────────────────────────────────────────────────────────
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
