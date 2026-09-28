package kordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// cleanupPendingAnnotation records when the cleanup conditions were first met.
// Used to honour deleteAfter grace periods across reconcile cycles.
const cleanupPendingAnnotation = "orkestra.orkspace.io/cleanup-pending-since"

// evaluateCleanup checks operatorBox.runtime.cleanup conditions.
// triggered=true  → delete this cycle.
// waitFor>0       → grace period running; re-enqueue after waitFor.
// both zero       → conditions not yet met; reconcile runs normally.
func (k *Kontroller) evaluateCleanup(
	ctx context.Context,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
	resolver *orktmpl.Resolver,
) (triggered bool, waitFor time.Duration, err error) {
	if !box.HasCleanup() {
		return false, 0, nil
	}

	cleanup := box.EffectiveCleanup()
	data := resolver.Data()
	key := obj.GetNamespace() + "/" + obj.GetName()

	met := orktypes.EvaluateConditions(data, cleanup.When, cleanup.Or, resolver.TemplateEvaluator())
	if !met {
		// Conditions not met — clear any pending annotation so the clock resets
		// if the CR transitions back to a terminal state later.
		if ann := obj.GetAnnotations(); ann != nil {
			if _, hasPending := ann[cleanupPendingAnnotation]; hasPending {
				delete(ann, cleanupPendingAnnotation)
				obj.SetAnnotations(ann)
				if pErr := k.kube.PatchAnnotations(ctx, obj, ann, metav1.PatchOptions{}); pErr != nil {
					logger.Warn().Err(pErr).
						Str("key", key).
						Msg("cleanup: failed to clear pending annotation")
				}
			}
		}
		return false, 0, nil
	}

	// Conditions are met. Apply deleteAfter grace period if declared.
	if cleanup.DeleteAfter.Duration > 0 {
		ann := obj.GetAnnotations()
		if ann == nil {
			ann = make(map[string]string)
		}
		pendingSince, hasPending := ann[cleanupPendingAnnotation]
		if !hasPending {
			// First time conditions are met — stamp the annotation.
			// Return the full deleteAfter as waitFor so the caller schedules
			// a re-enqueue via AddAfter.
			ann[cleanupPendingAnnotation] = time.Now().UTC().Format(time.RFC3339)
			obj.SetAnnotations(ann)
			if pErr := k.kube.PatchAnnotations(ctx, obj, ann, metav1.PatchOptions{}); pErr != nil {
				return false, 0, fmt.Errorf("cleanup: stamping pending annotation: %w", pErr)
			}
			logger.Info().
				Str("key", key).
				Str("deleteAfter", cleanup.DeleteAfter.Duration.String()).
				Msg("cleanup: conditions met — grace period started")
			return false, cleanup.DeleteAfter.Duration, nil
		}
		since, parseErr := time.Parse(time.RFC3339, pendingSince)
		if parseErr != nil {
			// Unparseable timestamp treated as elapsed.
			return true, 0, nil
		}
		if remaining := cleanup.DeleteAfter.Duration - time.Since(since); remaining > 0 {
			return false, remaining, nil
		}
	}

	return true, 0, nil
}

// deleteCR issues a foreground delete for obj. Called by the worker after
// maintain has removed deletion-protection labels (ForCleanup=true path).
// No-ops when the CR already has a deletionTimestamp (deletion in progress)
func (k *Kontroller) deleteCR(ctx context.Context, obj domain.Object) error {
	if !obj.GetDeletionTimestamp().IsZero() {
		return nil
	}

	name := obj.GetName()
	ns := obj.GetNamespace()
	key := ns + "/" + name

	propagation := metav1.DeletePropagationForeground
	if err := k.kube.Delete(ctx, obj, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}); err != nil {
		return fmt.Errorf("cleanup: deleting CR %s/%s: %w", ns, name, err)
	}
	logger.Info().
		Str("key", key).
		Msg("cleanup: CR deleted")
	return nil
}
