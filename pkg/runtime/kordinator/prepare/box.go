package prepare

import (
	"slices"

	orktarget "github.com/orkspace/orkestra/pkg/intent/target"
	"github.com/orkspace/orkestra/pkg/labels"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// effectiveBoxAndTarget returns the operative OperatorBox and resolved target
// for this reconcile cycle. The target is read from the serve-target annotation
// on the CR; falls back to the CRD-level box for direct kubectl applies.
// The system CleanupFinalizer is always present in the returned box.
func effectiveBoxAndTarget(crd orktypes.CRDEntry, obj *unstructured.Unstructured) (orktypes.OperatorBoxConfig, string) {
	target := orktarget.ResolveTargetFromAnnotations(obj.GetAnnotations())
	box := *crd.EffectiveOperatorBox(target)
	// Ensure the system cleanup finalizer is always present in the effective list.
	// Write to whichever finalizer field is active so EffectiveFinalizers() reflects the update.
	if box.Runtime != nil {
		if !slices.Contains(box.Runtime.Finalizers, labels.CleanupFinalizer) {
			box.Runtime.Finalizers = append(box.Runtime.Finalizers, labels.CleanupFinalizer)
		}
	} else {
		if box.Runtime == nil {
			box.Runtime = &orktypes.RuntimeConfig{}
		}
		if !slices.Contains(box.Runtime.Finalizers, labels.CleanupFinalizer) {
			box.Runtime.Finalizers = append(box.Runtime.Finalizers, labels.CleanupFinalizer)
		}
	}
	return box, target
}
