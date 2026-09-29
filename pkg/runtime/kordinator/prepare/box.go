package prepare

import (
	orktarget "github.com/orkspace/orkestra/pkg/intent/target"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// effectiveBoxAndTarget returns the operative OperatorBox and resolved target
// for this reconcile cycle. The target is read from the serve-target annotation
// on the CR; falls back to the CRD-level box for direct kubectl applies.
func effectiveBoxAndTarget(crd orktypes.CRDEntry, obj *unstructured.Unstructured) (orktypes.OperatorBoxConfig, string) {
	target := orktarget.ResolveTargetFromAnnotations(obj.GetAnnotations())
	box := *crd.EffectiveOperatorBox(target)
	return box, target
}
