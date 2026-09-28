package kordinator

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/post"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/prepare"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/vitals"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// PostInput holds everything runPost needs from the worker.
type PostInput struct {
	Kube      kubeclient.Interface
	Event     event.Recorder
	Health    *vitals.CRDHealth
	CRD       orktypes.CRDEntry
	Prepared  *domain.PreparedRequest
	Result    domain.Result
	ValResult *post.ValidationResult
}

// runPost runs the full post-reconcile sequence and returns the final reconcile error.
// Status patch and declarative event emission via post.Apply.
func runPost(ctx context.Context, in PostInput, reconcileErr error) error {
	box := prepare.BoxFrom(in.Prepared)
	resolver := in.Prepared.Context.(*orktmpl.Resolver)
	post.Apply(ctx, post.Input{
		CRD:         in.CRD,
		Kube:        in.Kube,
		Recorder:    in.Event,
		MetricsMap:  in.Health.GetAutoMetrics(),
		HealthMap:   in.Health.HealthAsMap(),
		StatusPatch: in.Result.StatusPatch,
	}, in.Prepared.Object, resolver, box, reconcileErr, in.ValResult)

	return reconcileErr
}
