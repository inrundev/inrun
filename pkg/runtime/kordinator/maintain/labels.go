package maintain

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
)

// applyLabels applies the three system label invariants plus any user-defined
// labels to obj, then patches labels and annotations in two separate calls.
//
// Label invariants:
//  1. managed=true           — ownership marker, always present
//  2. deletion-protection    — present iff global protection + CRD protectCRs
//  3. strict-mode-exempt     — present iff the CRD has opted out of strictMode
//
// Two-phase protection removal: transitioning protected→unprotected forces the
// strict-mode-exempt label ON for one extra cycle so the strict-mode webhook
// permits the deletion-protection label removal. On the next reconcile the
// exempt label is cleaned up without webhook interference.
func applyLabels(ctx context.Context, in Input, obj domain.Object, resolver *orktmpl.Resolver) error {
	mgr := labels.NewManager(labels.Config{
		Standalone:                in.Kat.IsStandaloneGateway(),
		DeletionProtectionEnabled: in.Kat.IsDeletionProtectionEnabled(),
	})

	serverLabels := copyStringMap(obj.GetLabels()) // snapshot before mutation

	mgr.EnsureManagedLabel(obj)

	if in.Kat.IsDeletionProtectionEnabled() {
		// ForCleanup overrides protection so the two-phase label removal runs
		// before the kordinator issues the Delete call on this same cycle.
		shouldHaveProtection := !in.ForCleanup && in.Kat.IsDeletionProtectionEnabled() && in.CRD.ShouldProtectCRs()
		mgr.EnsureDeletionProtectionLabel(obj, shouldHaveProtection)

		effectiveStrict := in.CRD.IsStrictDeletionProtection(in.Kat.IsStrictModeEnabled())
		currentlyProtected := serverLabels[labels.DeletionProtectionLabel] == labels.DeletionProtectionValue
		if !shouldHaveProtection && currentlyProtected {
			effectiveStrict = false
		}

		logger.Debug().
			Str("crd", in.CRD.Name).
			Str("resource", obj.GetName()).
			Bool("effectiveStrict", effectiveStrict).
			Bool("currentlyProtected", currentlyProtected).
			Msg("label: evaluating strict mode")
		mgr.EnsureStrictModeExemptLabel(obj, effectiveStrict)
	}

	if in.CRD.HasUserLabels() {
		resolved := make(map[string]string, len(in.CRD.Labels))
		for k, v := range in.CRD.Labels {
			val, err := resolver.Resolve(v)
			if err != nil {
				return fmt.Errorf("labels: CRD %q: key %q: %w", in.CRD.Name, k, err)
			}
			resolved[k] = val
		}
		mgr.EnsureUserLabels(obj, resolved)
	}

	if err := in.Kube.PatchLabels(ctx, obj, serverLabels, obj.GetLabels(), metav1.PatchOptions{}); err != nil {
		return err
	}

	if in.CRD.HasUserAnnotations() {
		resolved := make(map[string]string, len(in.CRD.Annotations))
		for k, v := range in.CRD.Annotations {
			val, err := resolver.Resolve(v)
			if err != nil {
				return fmt.Errorf("annotations: CRD %q: key %q: %w", in.CRD.Name, k, err)
			}
			resolved[k] = val
		}
		mgr.EnsureUserAnnotations(obj, resolved)
	}

	if mgr.EnsureManagedAnnotations(obj, in.CRD.KatalogName) {
		if err := in.Kube.PatchAnnotations(ctx, obj, obj.GetAnnotations(), metav1.PatchOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
