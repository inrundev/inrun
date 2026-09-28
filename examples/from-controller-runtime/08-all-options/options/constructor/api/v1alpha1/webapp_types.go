//go:build ignore

// api/v1alpha1/webapp_types.go
//
// The Go type for the ConstructorApp CRD.
// Used by the constructor reconciler — gives type-safe access to spec fields.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group    = "allopt.demo.orkestra.io"
	Version  = "v1alpha1"
	Kind     = "ConstructorApp"
	Resource = "constructorapps"
)

var (
	GroupVersion         = schema.GroupVersion{Group: Group, Version: Version}
	GroupVersionKind     = schema.GroupVersionKind{Group: Group, Version: Version, Kind: Kind}
	GroupVersionResource = schema.GroupVersionResource{
		Group:    Group,
		Version:  Version,
		Resource: Resource,
	}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// WebAppSpec defines the desired state of ConstructorApp.
type WebAppSpec struct {
	Image    string `json:"image"`
	Replicas int32  `json:"replicas,omitempty"`
	Port     int32  `json:"port,omitempty"`
}

// WebAppStatus defines the observed state of ConstructorApp.
type WebAppStatus struct {
	Phase      string             `json:"phase,omitempty"`
	Endpoint   string             `json:"endpoint,omitempty"`
	Replicas   int32              `json:"replicas,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ConstructorApp is the Schema for the webapps API.
type ConstructorApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebAppSpec   `json:"spec,omitempty"`
	Status WebAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConstructorAppList contains a list of ConstructorApp.
type ConstructorAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConstructorApp `json:"items"`
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion, &ConstructorApp{}, &ConstructorAppList{})
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}

func (w *ConstructorApp) DeepCopyObject() runtime.Object {
	if w == nil {
		return nil
	}
	out := new(ConstructorApp)
	w.DeepCopyInto(out)
	return out
}

func (w *ConstructorApp) DeepCopyInto(out *ConstructorApp) {
	*out = *w
	out.TypeMeta = w.TypeMeta
	w.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = w.Spec
	w.Status.DeepCopyInto(&out.Status)
}

func (s *WebAppStatus) DeepCopyInto(out *WebAppStatus) {
	*out = *s
	if s.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(s.Conditions))
		copy(out.Conditions, s.Conditions)
	}
}

func (wl *ConstructorAppList) DeepCopyObject() runtime.Object {
	if wl == nil {
		return nil
	}
	out := new(ConstructorAppList)
	*out = *wl
	if wl.Items != nil {
		out.Items = make([]ConstructorApp, len(wl.Items))
		for i := range wl.Items {
			wl.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
	return out
}
