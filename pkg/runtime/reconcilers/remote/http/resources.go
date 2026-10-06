package http

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inrundev/inrun/domain"
	"github.com/inrundev/inrun/pkg/kubeclient"
	"github.com/inrundev/inrun/pkg/labels"
	"github.com/inrundev/inrun/pkg/resources/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const remoteFieldManager = "inrun-remote"

// remoteResource pairs a parsed Kubernetes object with its optional per-resource
// forceConflict override, extracted from the raw map before SSA.
type remoteResource struct {
	obj           *unstructured.Unstructured
	forceConflict *bool
}

// apply validates and SSA-applies each resource returned by the remote reconciler.
//
// All resources are validated against managedResources before any are applied.
// Resources in the same namespace as the owner CR get an owner reference so
// Kubernetes GC cascades on CR deletion.
func (r *Reconciler) apply(ctx context.Context, owner domain.Object, resources []map[string]interface{}) error {
	if len(resources) == 0 {
		return nil
	}

	// Build lookup set of declared group+plural pairs.
	allowed := make(map[string]struct{}, len(r.managedResources))
	for _, mr := range r.managedResources {
		allowed[mr.Group+"/"+mr.Plural] = struct{}{}
	}

	// Parse and validate all resources before applying anything (fail fast).
	targets := make([]remoteResource, 0, len(resources))
	for i, raw := range resources {
		rr, err := parseResource(raw)
		if err != nil {
			return fmt.Errorf("resource[%d]: %w", i, err)
		}

		mapping, err := kubeclient.GVRFor(r.kube, rr.obj)
		if err != nil {
			return fmt.Errorf("resource[%d] %q (%s): no REST mapping — is the CRD installed? %w",
				i, rr.obj.GetName(), rr.obj.GroupVersionKind(), err)
		}

		gvr := mapping.Resource
		if _, ok := allowed[gvr.Group+"/"+gvr.Resource]; !ok {
			return fmt.Errorf("resource[%d] %q (%s/%s) was not declared in reconcile.remote.managedResources — add it to the catalog before the remote reconciler can create it",
				i, rr.obj.GetName(), gvr.Group, gvr.Resource)
		}

		targets = append(targets, rr)
	}

	// All resources validated — now apply.
	ownerRefs := shared.ResolveOwnerReferences(owner)
	ownerNS := owner.GetNamespace()
	catalogName, _ := owner.GetAnnotations()[labels.AnnotationManagedBy]

	for i, rr := range targets {
		if err := applySingleResource(ctx, r.kube, rr, ownerRefs, ownerNS, catalogName); err != nil {
			return fmt.Errorf("resource[%d] %q: apply failed: %w", i, rr.obj.GetName(), err)
		}
	}
	return nil
}

// parseResource extracts the optional forceConflict field from the raw map,
// removes it so Kubernetes never sees it, then unmarshals the rest.
func parseResource(raw map[string]interface{}) (remoteResource, error) {
	var fc *bool
	if v, ok := raw["forceConflict"]; ok {
		if b, ok := v.(bool); ok {
			fc = &b
		}
		delete(raw, "forceConflict")
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return remoteResource{}, fmt.Errorf("marshal: %w", err)
	}
	u := &unstructured.Unstructured{}
	if err := json.Unmarshal(data, &u.Object); err != nil {
		return remoteResource{}, fmt.Errorf("unmarshal: %w", err)
	}
	if u.GetName() == "" {
		return remoteResource{}, fmt.Errorf("resource has no metadata.name")
	}
	if u.GroupVersionKind().Empty() {
		return remoteResource{}, fmt.Errorf("resource %q has no apiVersion/kind", u.GetName())
	}
	return remoteResource{obj: u, forceConflict: fc}, nil
}

func applySingleResource(
	ctx context.Context,
	kube kubeclient.Interface,
	rr remoteResource,
	ownerRefs []metav1.OwnerReference,
	ownerNS string,
	catalogName string,
) error {
	u := rr.obj

	// Management labels and annotations — same invariants as the standard reconcile path.
	mgr := labels.NewManager(labels.Config{})
	mgr.EnsureManagedLabel(u)
	mgr.EnsureManagedAnnotations(u, catalogName)

	// Owner reference: same-namespace only.
	resNS := u.GetNamespace()
	if resNS != "" && resNS == ownerNS {
		u.SetOwnerReferences(ownerRefs)
	}

	resource, err := kubeclient.ResourceFor(kube, u)
	if err != nil {
		return fmt.Errorf("resolve resource: %w", err)
	}

	force := shared.ResolveForceConflict(kube, rr.forceConflict)
	_, err = resource.Apply(ctx, u.GetName(), u, metav1.ApplyOptions{
		FieldManager: remoteFieldManager,
		Force:        force != nil && *force,
	})
	return err
}
