package remote

import (
	"context"
	"fmt"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
)

// toResult converts a RemoteReconcileResult to scheduling intent and status patch.
func (r *RemoteReconciler) toResult(ctx context.Context, req domain.Request, res RemoteReconcileResult) (domain.Result, error) {
	switch res.Result {
	case "ok", "":
		return domain.Result{StatusPatch: res.Status}, nil

	case "requeue":
		if res.RequeueAfter == "" {
			return domain.Result{}, fmt.Errorf("remote reconciler: result=requeue but requeueAfter is empty")
		}
		d, err := time.ParseDuration(res.RequeueAfter)
		if err != nil {
			return domain.Result{}, fmt.Errorf("remote reconciler: invalid requeueAfter %q: %w", res.RequeueAfter, err)
		}
		logger.FromContext(ctx).Info().
			Str("reconciler", "remote").
			Str("key", req.Key).
			Str("requeue_after", res.RequeueAfter).
			Msg("remote reconciler requested requeue")
		return domain.Result{RequeueAfter: d, StatusPatch: res.Status}, nil

	case "error":
		msg := res.Error
		if msg == "" {
			msg = "remote reconciler returned result=error with no message"
		}
		logger.FromContext(ctx).Error().
			Str("reconciler", "remote").
			Str("key", req.Key).
			Str("endpoint", r.decl.Endpoint).
			Msg(msg)
		return domain.Result{}, fmt.Errorf("%s", msg)

	default:
		return domain.Result{}, fmt.Errorf("remote reconciler: unknown result value %q", res.Result)
	}
}

// resolveResources walks the resources array, expanding intent-form entries into full
// Kubernetes object maps. Full-form entries (with apiVersion) pass through unchanged.
func (r *RemoteReconciler) resolveResources(raw []map[string]interface{}, req domain.Request) ([]map[string]interface{}, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var owner domain.Object
	if req.Prepared != nil {
		owner = req.Prepared.Object
	}

	out := make([]map[string]interface{}, 0, len(raw))
	for i, entry := range raw {
		if _, hasAPI := entry["apiVersion"]; hasAPI {
			out = append(out, entry)
			continue
		}
		typVal, hasType := entry["type"]
		fieldsVal, hasFields := entry["fields"]
		if !hasType || !hasFields {
			return nil, fmt.Errorf("remote reconciler: resources[%d]: expected 'apiVersion' (full form) or 'type'+'fields' (intent form)", i)
		}
		typStr, ok := typVal.(string)
		if !ok {
			return nil, fmt.Errorf("remote reconciler: resources[%d]: 'type' must be a string", i)
		}
		fieldsMap, ok := fieldsVal.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("remote reconciler: resources[%d]: 'fields' must be an object", i)
		}
		built, err := buildFromIntent(typStr, fieldsMap, owner)
		if err != nil {
			return nil, fmt.Errorf("remote reconciler: resources[%d]: %w", i, err)
		}
		out = append(out, built)
	}
	return out, nil
}
