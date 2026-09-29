package shared

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ToObjectMap stamps apiVersion/kind onto obj and converts it to the
// map[string]interface{} form required by domain.Result.Resources and SSA apply.
// apiVersion is the full "group/version" string (e.g. "apps/v1", "v1").
func ToObjectMap(obj runtime.Object, apiVersion, kind string) (map[string]interface{}, error) {
	obj.GetObjectKind().SetGroupVersionKind(
		schema.FromAPIVersionAndKind(apiVersion, kind),
	)
	return runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
}

// DecodeFields JSON-round-trips a raw fields map into dst (must be a pointer).
// Used by Build functions to convert the intent fields map to a typed struct.
func DecodeFields(fields map[string]interface{}, dst interface{}) error {
	raw, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("decode fields: %w", err)
	}
	return nil
}
