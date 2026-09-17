package maintain

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// applyFinalizers adds or removes finalizers depending on CRDEntry.RemoveFinalizers.
//
// Normal operation (RemoveFinalizers=false): ensure all box.Finalizers are present.
// Force-cleanup mode (RemoveFinalizers=true): strip all box.Finalizers from the object.
func applyFinalizers(
	ctx context.Context,
	in Input,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	if in.CRD.RemoveFinalizers {
		return stripFinalizers(ctx, in, obj, box)
	}
	return ensureFinalizers(ctx, in, obj, box)
}

func ensureFinalizers(
	ctx context.Context,
	in Input,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	if len(box.Finalizers) == 0 {
		return nil
	}

	logger.Debug().
		Str("name", obj.GetName()).
		Any("crd finalizers", box.Finalizers).
		Msgf("checking finalizers: %v", obj.GetFinalizers())

	if !labels.EnsureFinalizers(obj, box.Finalizers) {
		return nil
	}

	logger.Debug().
		Str("name", obj.GetName()).
		Msgf("adding finalizers → %v", obj.GetFinalizers())

	if in.Recorder != nil {
		in.Recorder.Eventf(obj, corev1.EventTypeNormal, in.CRD.APITypes.Kind+"FinalizerAdded",
			fmt.Sprintf("Added finalizers to %s/%s", obj.GetNamespace(), obj.GetName()))
	}

	return in.Kube.PatchFinalizers(ctx, obj, obj.GetFinalizers(), metav1.PatchOptions{})
}

func stripFinalizers(
	ctx context.Context,
	in Input,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	if len(obj.GetFinalizers()) == 0 {
		return nil
	}

	if !labels.StripFinalizers(obj, box.Finalizers) {
		return nil
	}

	logger.Debug().
		Str("name", obj.GetName()).
		Msgf("removing finalizers → %v", obj.GetFinalizers())

	return in.Kube.PatchFinalizers(ctx, obj, obj.GetFinalizers(), metav1.PatchOptions{})
}
