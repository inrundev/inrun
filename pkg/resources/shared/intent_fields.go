package shared

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/validate/content"
)

// ApplyMetaOverrides merges caller-supplied labels and annotations into an
// already-built object map. Called by every BuildFromIntent after ToObjectMap
// so that intent-form resources can carry arbitrary metadata without each
// builder duplicating the merge logic.
//
// Fails fast with a descriptive error if any key is not a valid qualified name
// or any label value is not a valid Kubernetes label value. Annotation values
// have no syntax restriction and are not validated.
func ApplyMetaOverrides(obj map[string]interface{}, labels, annotations map[string]string) error {
	if len(labels) == 0 && len(annotations) == 0 {
		return nil
	}
	if err := validateLabelMap(labels); err != nil {
		return err
	}
	if err := validateAnnotationKeys(annotations); err != nil {
		return err
	}

	meta, _ := obj["metadata"].(map[string]interface{})
	if meta == nil {
		meta = make(map[string]interface{})
		obj["metadata"] = meta
	}
	if len(labels) > 0 {
		existing, _ := meta["labels"].(map[string]interface{})
		if existing == nil {
			existing = make(map[string]interface{})
		}
		for k, v := range labels {
			existing[k] = v
		}
		meta["labels"] = existing
	}
	if len(annotations) > 0 {
		existing, _ := meta["annotations"].(map[string]interface{})
		if existing == nil {
			existing = make(map[string]interface{})
		}
		for k, v := range annotations {
			existing[k] = v
		}
		meta["annotations"] = existing
	}
	return nil
}

func validateLabelMap(labels map[string]string) error {
	for k, v := range labels {
		if errs := content.IsLabelKey(k); len(errs) > 0 {
			return fmt.Errorf("invalid label key %q: %s", k, strings.Join(errs, "; "))
		}
		if errs := content.IsLabelValue(v); len(errs) > 0 {
			return fmt.Errorf("invalid label value for key %q: %s", k, strings.Join(errs, "; "))
		}
	}
	return nil
}

func validateAnnotationKeys(annotations map[string]string) error {
	for k := range annotations {
		if errs := content.IsLabelKey(k); len(errs) > 0 {
			return fmt.Errorf("invalid annotation key %q: %s", k, strings.Join(errs, "; "))
		}
	}
	return nil
}

// DeploymentIntentFields are the fields accepted in the remote reconciler
// intent form for a deployment resource.
type DeploymentIntentFields struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Replicas    int32             `json:"replicas"`
	Port        int32             `json:"port"`
	Env         map[string]string `json:"env"`
	Command     []string          `json:"command"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// ServiceIntentFields are the fields accepted in the remote reconciler
// intent form for a service resource.
type ServiceIntentFields struct {
	Name        string            `json:"name"`
	Port        int32             `json:"port"`
	TargetPort  int32             `json:"targetPort"`
	Type        string            `json:"type"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// ConfigMapIntentFields are the fields accepted in the remote reconciler
// intent form for a configmap resource.
type ConfigMapIntentFields struct {
	Name        string            `json:"name"`
	Data        map[string]string `json:"data"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// SecretIntentFields are the fields accepted in the remote reconciler
// intent form for a secret resource.
// Data values must be base64-encoded; StringData values are plaintext.
type SecretIntentFields struct {
	Name        string            `json:"name"`
	Data        map[string]string `json:"data"`
	StringData  map[string]string `json:"stringData"`
	SecretType  string            `json:"secretType"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// ServiceAccountIntentFields are the fields accepted in the remote reconciler
// intent form for a serviceaccount resource.
type ServiceAccountIntentFields struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// CustomResourceIntentFields are the fields accepted in the remote reconciler
// intent form for a custom resource (type: custom).
// The spec is schema-agnostic; any top-level fields beyond spec must use full form.
type CustomResourceIntentFields struct {
	APIVersion  string                 `json:"apiVersion"`
	Kind        string                 `json:"kind"`
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace,omitempty"`
	Spec        map[string]interface{} `json:"spec,omitempty"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
}

// IngressIntentFields are the fields accepted in the remote reconciler
// intent form for an ingress resource.
type IngressIntentFields struct {
	Name         string            `json:"name"`
	Host         string            `json:"host"`
	ServiceName  string            `json:"serviceName"`
	ServicePort  int32             `json:"servicePort"`
	Path         string            `json:"path"`
	PathType     string            `json:"pathType"`
	IngressClass string            `json:"ingressClass"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
}

// StatefulSetIntentFields are the fields accepted in the remote reconciler
// intent form for a statefulset resource.
type StatefulSetIntentFields struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Replicas    int32             `json:"replicas"`
	Port        int32             `json:"port"`
	ServiceName string            `json:"serviceName"`
	Env         map[string]string `json:"env"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// JobIntentFields are the fields accepted in the remote reconciler
// intent form for a job resource.
type JobIntentFields struct {
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	Command      []string          `json:"command"`
	Args         []string          `json:"args"`
	BackoffLimit int               `json:"backoffLimit"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
}

// CronJobIntentFields are the fields accepted in the remote reconciler
// intent form for a cronjob resource.
type CronJobIntentFields struct {
	Name              string            `json:"name"`
	Image             string            `json:"image"`
	Schedule          string            `json:"schedule"`
	Command           []string          `json:"command"`
	Args              []string          `json:"args"`
	ConcurrencyPolicy string            `json:"concurrencyPolicy"`
	Suspend           bool              `json:"suspend"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
}
