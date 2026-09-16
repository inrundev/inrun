package post

import (
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/utils/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// injectRuntimeAnnotations writes live metrics and health data as annotations
// on the CR so the gateway can use them for metrics-level and health-level gating
// in validation and mutation rules as well as preReconcile gates.
//
// Metrics injection respects crossAccess; health injection respects the health endpoint flag.
func injectRuntimeAnnotations(obj domain.Object, in Input) {
	u := toUnstructured(obj)
	if u == nil {
		return
	}
	if in.CRD.CrossAccessEnabled() && len(in.MetricsMap) > 0 {
		common.InjectMetricsAnnotation(u, in.MetricsMap)
	}
	if in.CRD.Endpoints.IsHealthEnabled() && len(in.HealthMap) > 0 {
		common.InjectHealthAnnotation(u, in.HealthMap)
	}
}

func toUnstructured(obj domain.Object) *unstructured.Unstructured {
	res, ok := domain.ToUnstructured(obj)
	if !ok {
		return nil
	}
	return res
}
