package post

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	orklabels "github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// injectRuntimeAnnotations writes live metrics and health data as annotations
// on the CR so the gateway can use them for metrics-level and health-level gating
// in validation and mutation rules as well as preReconcile gates, then patches
// them to the cluster. Patch failures are logged and do not fail the post phase.
//
// Metrics injection respects crossAccess; health injection respects the health endpoint flag.
func injectRuntimeAnnotations(ctx context.Context, obj domain.Object, in Input) {
	u := toUnstructured(obj)
	if u == nil {
		return
	}
	existing := u.GetAnnotations()
	before := [2]string{
		existing[orklabels.AnnotationCrossMetrics],
		existing[orklabels.AnnotationHealth],
	}

	if in.CRD.CrossAccessEnabled() && len(in.MetricsMap) > 0 {
		common.InjectMetricsAnnotation(u, in.MetricsMap)
	}
	if in.CRD.Endpoints.IsHealthEnabled() && len(in.HealthMap) > 0 {
		common.InjectHealthAnnotation(u, in.HealthMap)
	}

	after := u.GetAnnotations()
	if after[orklabels.AnnotationCrossMetrics] == before[0] &&
		after[orklabels.AnnotationHealth] == before[1] {
		return
	}
	if err := in.Kube.PatchAnnotations(ctx, obj, after, metav1.PatchOptions{}); err != nil {
		logger.FromContext(ctx).Warn().Err(err).
			Str("name", obj.GetName()).
			Msg("post: runtime annotation patch failed")
	}
}

func toUnstructured(obj domain.Object) *unstructured.Unstructured {
	res, ok := domain.ToUnstructured(obj)
	if !ok {
		return nil
	}
	return res
}
