package intent

import (
	"github.com/inrundev/inrun/pkg/labels"
	"github.com/inrundev/inrun/pkg/utils/common"
)

// Target extracts the effective target from a CR's annotations.
// Resolution order:
//  1. serve-alias annotation (most specific)
//  2. serve-target annotation (primary target)
//  3. Empty string (no target found)
func Target(annotations map[string]string) string {
	if annotations == nil {
		return ""
	}

	// 1. Check alias first (most specific)
	if alias, ok := annotations[labels.AnnotationServeAlias]; ok && alias != "" {
		return alias
	}

	// 2. Fall back to target
	if target, ok := annotations[labels.AnnotationServeTarget]; ok && target != "" {
		return target
	}

	return ""
}

// FromObject extracts the raw intent payload from the
// inrun.dev/serve-intent annotation on a CR object map.
// Returns nil when the annotation is absent or unparseable.
// Used by both the webhook and the reconciler to inject .request into
// the resolver so validation rules can reference intent-vocabulary fields.
func FromObject(obj map[string]interface{}) map[string]interface{} {
	return common.ResolveAnnotationFromObject(obj, labels.AnnotationServeIntent)
}
