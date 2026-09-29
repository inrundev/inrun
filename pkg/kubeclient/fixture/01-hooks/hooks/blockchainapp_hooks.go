package hooks

import (
	"context"
	"fmt"

	apiv1 "github.com/orkspace/orkestra-args-hooks/api/v1alpha1"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orkdeploy "github.com/orkspace/orkestra/pkg/resources/deployments"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BlockchainAppHooks returns the hook implementation registered in the Katalog.
func BlockchainAppHooks() domain.AnyReconcileHooks {
	return domain.ReconcileHooks[*apiv1.BlockchainApp]{OnReconcile: onBlockchainAppReconcile}
}

func onBlockchainAppReconcile(ctx context.Context, obj *apiv1.BlockchainApp) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not in context")
	}

	// Both values come from the Katalog — not from the CR spec.
	// inBusinessHours: computed from a user-defined note (world state).
	// featureEnabled:  result of an external HTTP call the runtime made,
	//                  but only when inBusinessHours was true (see katalog.yaml when:).
	// The hook binary never changes when the schedule or flag endpoint changes.
	inBusinessHours := kube.Args().String("inBusinessHours") == "true"
	featureEnabled := kube.Args().String("featureEnabled") == "true"

	annotation := "false"
	if inBusinessHours && featureEnabled {
		annotation = "true"
	}

	replicas := int32(obj.Spec.Replicas)
	if replicas == 0 {
		replicas = 1
	}

	spec := orkdeploy.ResolvedDeploymentSpec{
		Name:      obj.Name,
		Namespace: obj.Namespace,
		Image:     obj.Spec.Image,
		Replicas:  replicas,
		Annotations: map[string]string{
			"feature.demo/v2-enabled": annotation,
		},
	}
	if err := orkdeploy.Apply(ctx, kube, obj, spec); err != nil {
		return fmt.Errorf("blockchainapp deployment: %w", err)
	}

	return kube.PatchStatus(ctx, obj, map[string]any{
		"featureEnabled":  annotation,
		"inBusinessHours": inBusinessHours,
	}, metav1.PatchOptions{})
}
