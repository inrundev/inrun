// Package post runs the postReconcile phase: status patch and declarative event emission.
// Called by Kordinator after the reconciler returns, whether or not it errored.
// All operations are best-effort — failures are logged and do not requeue.
package post

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/children"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ValidationResult is the subset of prepare.ValidationResult that post needs
// for writing the ValidationFailed/ValidationWarning conditions.
type ValidationResult struct {
	Deny       bool
	Violations []ValidationViolation
	Warnings   []ValidationViolation
}

// ValidationViolation mirrors prepare.ValidationViolation for status conditions.
type ValidationViolation struct {
	Field   string
	Rule    string
	Value   string
	Message string
}

// Input holds everything Apply needs.
type Input struct {
	CRD      orktypes.CRDEntry
	Kube     kubeclient.Interface
	Recorder event.Recorder
	// MetricsMap carries live queue/worker/autoscale metrics for resolver enrichment
	// and metrics annotation injection. Nil is safe — enrichment is skipped.
	MetricsMap map[string]interface{}
	// HealthMap carries live CRD health data for resolver enrichment and health
	// annotation injection. Nil is safe — enrichment is skipped.
	HealthMap map[string]interface{}
}

// Apply runs the full post-reconcile phase.
//
// reconcileErr is nil on success; it drives the Ready condition and layer-2 field gating.
// valResult may be nil — when nil, validation conditions are written as Passed.
func Apply(
	ctx context.Context,
	in Input,
	obj domain.Object,
	resolver *orktmpl.Resolver,
	box orktypes.OperatorBoxConfig,
	reconcileErr error,
	valResult *ValidationResult,
) {
	// Extend resolver with child state for status field templates.
	if reconcileErr == nil && (box.OnCreate != nil || box.OnReconcile != nil) {
		ch := children.ReadChildren(ctx, in.Kube, obj, resolver, in.CRD)
		resolver = resolver.WithChildren(ch)
	}

	// Enrich resolver with live runtime metrics and health so status.fields
	// templates can reference .metrics.queueDepth, .health.state, etc.
	if len(in.MetricsMap) > 0 {
		resolver = resolver.WithMetrics(in.MetricsMap)
	}
	if len(in.HealthMap) > 0 {
		resolver = resolver.WithHealth(in.HealthMap)
	}

	// Annotate the object with runtime metrics/health
	// for gateway and preReconcile gating.
	injectRuntimeAnnotations(obj, in)

	applyStatus(ctx, in, obj, resolver, reconcileErr, valResult, box)
	applyEmit(ctx, in, obj, resolver, box, reconcileErr)
}
