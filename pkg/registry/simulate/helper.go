package simulate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	orktarget "github.com/orkspace/orkestra/pkg/intent/target"
	orklabels "github.com/orkspace/orkestra/pkg/labels"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

// ErrRemoteReconciler is returned by Run and RunWithEnvtest when the requested
// CRD declares reconcile.remote:. Remote reconcilers dispatch to an external
// HTTP endpoint at runtime — there is nothing to simulate in-process.
var ErrRemoteReconciler = errors.New("remote reconciler: cannot simulate — dispatches to an external endpoint at runtime")

// effectiveOperatorBox returns the effective operatorBox for a given CR.
func effectiveOperatorBox(entry orktypes.CRDEntry, cr *unstructured.Unstructured, target string) *orktypes.OperatorBoxConfig {
	if target != "" {
		return entry.EffectiveOperatorBox(target)
	}

	effectiveTarget := orktarget.ResolveTargetFromAnnotations(cr.GetAnnotations())
	return entry.EffectiveOperatorBox(effectiveTarget)
}

// getFromIndexerOrFallback returns a func that fetches CR from indexer or returns fallback.
func getFromIndexerOrFallback(indexer cache.Indexer, key string, cr *unstructured.Unstructured) func() *unstructured.Unstructured {
	return func() *unstructured.Unstructured {
		item, exists, _ := indexer.GetByKey(key)
		if !exists || item == nil {
			return nil
		}
		if u, ok := item.(*unstructured.Unstructured); ok {
			return u
		}
		return cr
	}
}

// resolveCacheKey returns the cache key for a CR.
func resolveCacheKey(cr *unstructured.Unstructured) (string, error) {
	key, err := cache.MetaNamespaceKeyFunc(cr)
	if err != nil {
		return "", fmt.Errorf("computing CR key: %w", err)
	}
	return key, nil
}

// parseCollectionURL returns (resource, namespace) from a collection-level URL
// (the target of a POST create where no name appears in the path).
//
//	/api/v1/namespaces/{ns}/{resource}
//	/api/v1/{resource}
//	/apis/{group}/{version}/namespaces/{ns}/{resource}
//	/apis/{group}/{version}/{resource}
func parseCollectionURL(req *http.Request) (resource, namespace string) {
	p := strings.TrimPrefix(req.URL.Path, "/")
	parts := strings.Split(p, "/")
	switch {
	case len(parts) == 5 && parts[0] == "api" && parts[2] == "namespaces":
		return parts[4], parts[3]
	case len(parts) == 3 && parts[0] == "api":
		return parts[2], ""
	case len(parts) == 6 && parts[0] == "apis" && parts[3] == "namespaces":
		return parts[5], parts[4]
	case len(parts) == 4 && parts[0] == "apis":
		return parts[3], ""
	}
	return
}

// extractMetaName reads metadata.name from a raw Kubernetes API response body.
func extractMetaName(body []byte) string {
	var obj struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(body, &obj); err == nil {
		return obj.Metadata.Name
	}
	return ""
}

// seedManagedMeta pre-populates managed labels and annotations on the CR so the
// reconciler's idempotency guards skip those patches in every cycle.
func seedManagedMeta(cr *unstructured.Unstructured, katalogName string) {
	labels := cr.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[orklabels.ManagedKey] = orklabels.ManagedValue
	labels[orklabels.DeletionProtectionLabel] = orklabels.DeletionProtectionValue
	cr.SetLabels(labels)

	ann := cr.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	ann[orklabels.AnnotationManagedBy] = katalogName
	ann[orklabels.AnnotationManagedSince] = time.Now().UTC().Format(time.RFC3339)
	cr.SetAnnotations(ann)
}

// opsMatch returns true when two op slices have the same verb+resource sequence.
func opsMatch(a, b []Op) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Verb != b[i].Verb || a[i].Resource != b[i].Resource {
			return false
		}
	}
	return true
}
