package utils

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// Wait for CRD creation
func WaitForCRD(cfg *rest.Config, group, kind, version string) error {
	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(
		memory.NewMemCacheClient(disco),
	)

	gk := schema.GroupKind{Group: group, Kind: kind}

	_, err = mapper.RESTMapping(gk, version)
	if meta.IsNoMatchError(err) {
		return fmt.Errorf("CRD %s.%s/%s not installed", kind, group, version)
	}
	return err
}

func RequireStrParams(required map[string]string) error {
	var missing []string
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		err := fmt.Sprintf("missing required parameter(s): %s", strings.Join(missing, ", "))
		return fmt.Errorf("%s", err)
	}
	return nil
}

// BuildAnnotationPatch constructs a JSON merge patch that sets one annotation.
func BuildAnnotationPatch(key, value string) map[string]interface{} {
	return map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				key: value,
			},
		},
	}
}

// Used to merge labels
func Merge(target *string, incoming, sep string) {
	if *target == "" {
		*target = incoming
	} else {
		*target = *target + sep + incoming
	}
}

// IsRunningInPod returns true when running inside a Kubernetes pod.
// The service account token is always present inside a pod.
func IsRunningInPod() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return err == nil
}
