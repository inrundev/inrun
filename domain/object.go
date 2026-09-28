package domain

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

// Object represents a Kubernetes API object with metadata and runtime type information.
type Object interface {
	metav1.Object
	runtime.Object
}

// ObjectList represents a Kubernetes API list object containing metadata
// and runtime type information.
type ObjectList interface {
	metav1.ListInterface
	runtime.Object
}

// ObjectKey identifies a Kubernetes object by namespace and name.
type ObjectKey = types.NamespacedName

// ToDomainObject unwraps a cache tombstone and asserts to domain.Object.
func ToDomainObject(obj interface{}) (Object, bool) {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	d, ok := obj.(Object)
	return d, ok
}

// UnwrapCacheTombstone unwraps a cache tombstone and returns the object.
func UnwrapCacheTombstone(obj interface{}) interface{} {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	return obj
}

// Raw returns the object's underlying field map without copying.
// Callers that need to mutate must call DeepCopy first.
func Raw(obj Object) map[string]interface{} {
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u.Object
	}
	return nil
}

// ToUnstructured unwraps a cache tombstone and converts to *unstructured.Unstructured.
// Works for both dynamic informers (which store *unstructured.Unstructured) and typed
// informers (which store scheme-registered concrete types such as *v1alpha1.WebApp).
func ToUnstructured(obj interface{}) (*unstructured.Unstructured, bool) {
	if ts, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = ts.Obj
	}
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u, true
	}
	ro, ok := obj.(runtime.Object)
	if !ok {
		return nil, false
	}
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ro)
	if err != nil {
		return nil, false
	}
	return &unstructured.Unstructured{Object: m}, true
}
