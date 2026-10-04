// Package prepare builds the domain.PreparedRequest for a single reconcile cycle.
// Called by Kordinator after the reconcile gate passes, before the reconciler is invoked.
package prepare

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	orkexternal "github.com/orkspace/orkestra/pkg/external"
	"github.com/orkspace/orkestra/pkg/intent"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/contract"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// Input holds everything Prepare needs from the Kontroller.
type Input struct {
	Entry    contract.RegistryEntry
	Key      string
	Kat      *katalog.Katalog
	Kube     kubeclient.Interface
	Registry KatalogRegistry
}

// Prepare builds the PreparedRequest for a single reconcile cycle.
//
// Steps:
//  1. Object from cache
//  2. GVK fix
//  3. Namespace guard — returns (nil, nil, nil) for restricted/non-allowed namespaces
//  4. Effective box + target
//  5. Normalize + base resolver
//  6. Enrichment chain (profiles, notes, intent, uniqueness, hooks external)
//  7. Cross-CRD observation
//  8. Mutation (if mutateFirst)
//  9. Validation / admission
//  10. Mutation (if !mutateFirst)
//
// Returns (nil, nil, nil) when the object is not in cache (deleted between dequeue
// and preparation) or when the namespace guard blocks it.
// Returns (nil, nil, err) on failure — caller should requeue.
func Prepare(ctx context.Context, in Input) (*domain.PreparedRequest, *ValidationResult, error) {
	if in.Entry.Informer == nil {
		return nil, nil, nil
	}

	// Step 1: object from cache
	raw, exists, err := in.Entry.Informer.GetIndexer().GetByKey(in.Key)
	if err != nil || !exists || raw == nil {
		return nil, nil, nil
	}
	obj, ok := domain.ToUnstructured(raw)
	if !ok {
		return nil, nil, fmt.Errorf("prepare: unexpected type %T in informer cache for %q", raw, in.Key)
	}

	// Step 2: GVK fix — watch events sometimes omit TypeMeta on first reconcile
	if obj.GetObjectKind().GroupVersionKind().Empty() {
		obj.GetObjectKind().SetGroupVersionKind(in.Entry.CRD.GVK())
	}

	crd := in.Entry.CRD

	// Step 3: namespace guard — skip reconcile for restricted/non-allowed namespaces.
	// Deletion is always permitted so finalizers can be removed;
	// this guard runs only for non-deleting CRs.
	if crd.HasNamespaceRules() && obj.GetDeletionTimestamp() == nil {
		result := CheckNamespace(ctx, obj, obj.GetNamespace(), crd.AllRestrictedNamespaces(), crd.AllAllowedNamespaces(), crd.APITypes.Kind)
		if !result.Allowed {
			return nil, nil, nil
		}
	}

	// Step 4: effective box and target
	box, target := effectiveBoxAndTarget(crd, obj)

	// Step 5: normalize + base resolver
	normalized, resolver, normalizeChanges, err := applyNormalize(ctx, crd, obj)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare: %w", err)
	}
	if len(normalizeChanges) > 0 {
		resolver = resolver.WithNormalizeChanges(normalizeChanges)
	}

	// Step 6: enrichment chain
	if in.Kat != nil {
		if !in.Kat.Profiles.Empty() {
			resolver = resolver.WithProfiles(&in.Kat.Profiles)
		}
		if !in.Kat.Notes.Empty() {
			resolver = resolver.WithUserNotes(in.Kat.UserNotes())
		}
	}
	if req := intent.FromObject(resolver.Data()); req != nil {
		resolver = resolver.WithRequest(req)
	}
	if in.Kube != nil {
		resolver = resolver.WithUniquenessChecker(
			newUniquenessChecker(ctx, in.Kube, crd.GVR(), crd.IsNamespaced()),
		)
	}

	// Hooks-declared external calls — runs before ScopedFor so results are
	// available as .external.<name>.* when hook args templates are evaluated.
	if crd.HasHooksExternal() {
		var extErr error
		resolver, extErr = orkexternal.Run(ctx, crd.GVKString(), resolver, crd.HooksExternal(), in.Kube.Clientset())
		if extErr != nil {
			return nil, nil, fmt.Errorf("prepare: hooks external: %w", extErr)
		}
	}

	// Step 7: cross-CRD observation
	if len(box.EffectiveCross()) > 0 && in.Registry != nil {
		cs := in.Kube.Clientset()
		crossData := readCross(ctx, obj, box.EffectiveCross(), resolver, in.Registry, cs)
		if len(crossData) > 0 {
			resolver = resolver.WithCross(crossData)
		}
	}

	// Steps 8–10: mutation / validation / admission
	// Order respects MutationConfig.MutateFirst:
	//   true (default) — mutate → validate → reconcile
	//   false          — validate → mutate → reconcile
	if crd.HasMutationRules() && crd.ShouldMutateFirst() {
		if resolver, err = applyMutation(ctx, in.Kube, normalized, resolver, crd); err != nil {
			// Mutation failures are non-fatal — log and continue
			_ = err
		}
	}

	var valResult *ValidationResult
	if crd.HasValidationRules() {
		var valErr error
		resolver, valResult, valErr = applyValidation(ctx, in.Kube, obj, resolver, crd)
		if valErr != nil {
			return nil, valResult, valErr // denial carries valResult for status write
		}
	}

	if crd.HasMutationRules() && !crd.ShouldMutateFirst() {
		if resolver, err = applyMutation(ctx, in.Kube, normalized, resolver, crd); err != nil {
			_ = err
		}
	}

	return &domain.PreparedRequest{
		Object:  normalized,
		Context: resolver,
		Target:  target,
		Box:     box,
	}, valResult, nil
}

// BoxFrom extracts the OperatorBoxConfig from a PreparedRequest.
// Only Kordinator constructs PreparedRequest — any other Box type is a programming error.
func BoxFrom(req *domain.PreparedRequest) orktypes.OperatorBoxConfig {
	if req == nil {
		return orktypes.OperatorBoxConfig{}
	}
	box, ok := req.Box.(orktypes.OperatorBoxConfig)
	if !ok {
		panic(fmt.Sprintf("prepare.BoxFrom: unexpected Box type %T", req.Box))
	}
	return box
}
