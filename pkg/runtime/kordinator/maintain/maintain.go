// Package maintain applies labels, annotations, and finalizers to a CR
// on every reconcile cycle — after prepare and before reconcile.
//
// It is a standalone package; it does not import the reconciler package.
package maintain

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// Input holds everything Apply needs from the Kontroller.
type Input struct {
	CRD      orktypes.CRDEntry
	Kat      *katalog.Katalog
	Kube     kubeclient.Interface
	Recorder event.Recorder

	// ForCleanup signals that the CR is about to be deleted by the cleanup gate.
	// When true, maintain sets shouldHaveProtection=false so the two-phase
	// deletion-protection label removal runs in this cycle — before the Delete call.
	ForCleanup bool
}

// Apply ensures finalizers, labels, and annotations are up to date.
// It issues Kubernetes patch calls for each category only when a diff is found.
func Apply(
	ctx context.Context,
	in Input,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
	resolver *orktmpl.Resolver,
) error {
	if err := applyFinalizers(ctx, in, obj, box); err != nil {
		return fmt.Errorf("maintain: finalizers: %w", err)
	}
	if err := applyLabels(ctx, in, obj, resolver); err != nil {
		return fmt.Errorf("maintain: labels: %w", err)
	}
	return nil
}
