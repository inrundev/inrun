package hooks

import (
	"context"
	"fmt"

	apiv1 "github.com/orkspace/orkestra-args-hooks-targets/api/v1alpha1"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orkdeploy "github.com/orkspace/orkestra/pkg/resources/deployments"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BlockchainAppHooks returns the hook implementation registered in the Katalog.
func BlockchainAppHooks() domain.AnyReconcileHooks {
	return domain.ReconcileHooks[*apiv1.BlockchainAppWithTargets]{OnReconcile: onBlockchainAppWithTargetsReconcile}
}

func onBlockchainAppWithTargetsReconcile(ctx context.Context, obj *apiv1.BlockchainAppWithTargets) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not in context")
	}

	// Both values come from the Katalog args — resolved per target surface.
	// v2-enabled target: featureEnabled="true", enqueueGate blocks outside hours.
	// v2-disabled target: featureEnabled="false", no gate.
	// The hook binary is identical — only the args change between targets.
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
		return fmt.Errorf("blockchainappwithtargets deployment: %w", err)
	}

	return kube.PatchStatus(ctx, obj, map[string]any{
		"featureEnabled":  annotation,
		"inBusinessHours": inBusinessHours,
	}, metav1.PatchOptions{})
}
