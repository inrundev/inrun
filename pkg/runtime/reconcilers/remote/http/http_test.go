package http

import (
	"testing"

	"github.com/orkspace/orkestra/domain"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func newReconcilerForTest(endpoint string) *Reconciler {
	return &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Endpoint: endpoint,
		},
	}
}

func reqWithResolver(data map[string]interface{}) domain.Request {
	resolver := orktmpl.NewResolverFromMap(data)
	return domain.Request{
		Prepared: &domain.PreparedRequest{Context: resolver},
	}
}

func TestResolveEndpoint_Static(t *testing.T) {
	r := newReconcilerForTest("http://svc.internal/reconcile")
	assert.Equal(t, "http://svc.internal/reconcile", r.resolveEndpoint(domain.Request{}))
}

func TestResolveEndpoint_Template(t *testing.T) {
	r := newReconcilerForTest("http://{{ index . \"namespace\" }}-svc/reconcile")
	req := reqWithResolver(map[string]interface{}{"namespace": "prod"})
	assert.Equal(t, "http://prod-svc/reconcile", r.resolveEndpoint(req))
}

func TestResolveEndpoint_NilPrepared_FallsBack(t *testing.T) {
	r := newReconcilerForTest("http://{{ .metadata.namespace }}-svc/reconcile")
	assert.Equal(t, "http://{{ .metadata.namespace }}-svc/reconcile", r.resolveEndpoint(domain.Request{}))
}

func TestResolveArgs_NoArgs(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	assert.Nil(t, r.resolveArgs(domain.Request{}))
}

func TestResolveArgs_StaticPassThrough(t *testing.T) {
	r := &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Endpoint: "http://svc/reconcile",
			Args:     map[string]interface{}{"env": "prod", "replicas": 3},
		},
	}
	got := r.resolveArgs(domain.Request{})
	assert.Equal(t, "prod", got["env"])
	assert.Equal(t, 3, got["replicas"])
}

func TestResolveArgs_TemplateCoercion(t *testing.T) {
	r := &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Endpoint: "http://svc/reconcile",
			Args: map[string]interface{}{
				"replicas": `{{ index . "replicas" }}`,
				"debug":    `{{ index . "debug" }}`,
			},
		},
	}
	req := reqWithResolver(map[string]interface{}{"replicas": "5", "debug": "true"})
	got := r.resolveArgs(req)
	assert.Equal(t, float64(5), got["replicas"])
	assert.Equal(t, true, got["debug"])
}

// ownerObj builds a minimal *unstructured.Unstructured for use as an owner in tests.
func ownerObj(name, namespace string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetAPIVersion("example.io/v1")
	u.SetKind("App")
	u.SetUID("test-uid")
	return u
}

func reqWithOwner(owner domain.Object) domain.Request {
	return domain.Request{
		Key: owner.GetNamespace() + "/" + owner.GetName(),
		Prepared: &domain.PreparedRequest{
			Object: owner,
		},
	}
}

func TestResolveResources_Empty(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	got, err := r.resolveResources(nil, domain.Request{})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestResolveResources_FullForm_PassThrough(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	full := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "my-app"},
	}
	got, err := r.resolveResources([]map[string]interface{}{full}, domain.Request{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "apps/v1", got[0]["apiVersion"])
}

func TestResolveResources_IntentForm_Deployment(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	req := reqWithOwner(ownerObj("my-app", "default"))
	intent := map[string]interface{}{
		"type": "deployment",
		"fields": map[string]interface{}{
			"name":  "my-app",
			"image": "nginx:1.25",
		},
	}
	got, err := r.resolveResources([]map[string]interface{}{intent}, req)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "apps/v1", got[0]["apiVersion"])
	assert.Equal(t, "Deployment", got[0]["kind"])
}

func TestResolveResources_IntentForm_UnknownType(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	req := reqWithOwner(ownerObj("my-app", "default"))
	intent := map[string]interface{}{
		"type":   "daemonset",
		"fields": map[string]interface{}{"name": "my-ds"},
	}
	_, err := r.resolveResources([]map[string]interface{}{intent}, req)
	assert.ErrorContains(t, err, "unknown resource type")
}

func TestResolveResources_IntentForm_NilOwner_Errors(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	intent := map[string]interface{}{
		"type":   "deployment",
		"fields": map[string]interface{}{"name": "my-app", "image": "nginx:1.25"},
	}
	_, err := r.resolveResources([]map[string]interface{}{intent}, domain.Request{})
	assert.ErrorContains(t, err, "prepared request")
}

func TestResolveResources_IntentForm_Custom(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	req := reqWithOwner(ownerObj("my-app", "default"))
	intent := map[string]interface{}{
		"type": "custom",
		"fields": map[string]interface{}{
			"apiVersion": "example.io/v1",
			"kind":       "WebApp",
			"name":       "my-app",
			"spec":       map[string]interface{}{"replicas": float64(3)},
		},
	}
	got, err := r.resolveResources([]map[string]interface{}{intent}, req)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "example.io/v1", got[0]["apiVersion"])
	assert.Equal(t, "WebApp", got[0]["kind"])
}

func TestResolveResources_IntentForm_Custom_MissingAPIVersion(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	req := reqWithOwner(ownerObj("my-app", "default"))
	intent := map[string]interface{}{
		"type": "custom",
		"fields": map[string]interface{}{
			"kind": "WebApp",
			"name": "my-app",
		},
	}
	_, err := r.resolveResources([]map[string]interface{}{intent}, req)
	assert.ErrorContains(t, err, "apiVersion is required")
}

func TestResolveResources_MixedForms(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	req := reqWithOwner(ownerObj("my-app", "default"))
	raw := []map[string]interface{}{
		{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]interface{}{"name": "cfg"}},
		{"type": "serviceaccount", "fields": map[string]interface{}{"name": "my-sa"}},
	}
	got, err := r.resolveResources(raw, req)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "v1", got[0]["apiVersion"])
	assert.Equal(t, "ServiceAccount", got[1]["kind"])
}

func TestApplyObjectExclusions_RemovesPaths(t *testing.T) {
	r := &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Endpoint: "http://svc/reconcile",
			Payload: &orktypes.RemotePayloadConfig{
				Object: &orktypes.RemotePayloadObjectConfig{
					Exclude: []string{"metadata.managedFields", "metadata.annotations"},
				},
			},
		},
	}
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":          "my-app",
			"managedFields": []interface{}{"some-data"},
			"annotations":   map[string]interface{}{"kubectl.io/last-applied": "{}"},
		},
		"spec": map[string]interface{}{"replicas": float64(2)},
	}}
	got := r.applyObjectExclusions(obj, r.decl.Payload.Object.Exclude)
	meta := got["metadata"].(map[string]interface{})
	assert.Equal(t, "my-app", meta["name"])
	assert.Nil(t, meta["managedFields"])
	assert.Nil(t, meta["annotations"])
	assert.NotNil(t, got["spec"])
}

func TestApplyObjectExclusions_NoExclusions_ReturnsOriginalMap(t *testing.T) {
	r := newReconcilerForTest("http://svc/reconcile")
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{"name": "my-app"},
	}}
	got := r.applyObjectExclusions(obj, nil)
	assert.Equal(t, "my-app", got["metadata"].(map[string]interface{})["name"])
}

func TestApplyObjectExclusions_DoesNotMutateOriginal(t *testing.T) {
	r := &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Endpoint: "http://svc/reconcile",
		},
	}
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":          "my-app",
			"managedFields": []interface{}{"data"},
		},
		"spec": map[string]interface{}{"image": "nginx:1.25"},
	}}
	_ = r.applyObjectExclusions(obj, []string{"metadata.managedFields"})
	meta := obj.Object["metadata"].(map[string]interface{})
	assert.NotNil(t, meta["managedFields"], "original object must not be mutated")
}

func TestApplyChildrenConfig_ResourcesFilter(t *testing.T) {
	r := &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Payload: &orktypes.RemotePayloadConfig{
				Children: &orktypes.RemotePayloadChildrenConfig{
					Resources: map[string]*orktypes.RemotePayloadChildrenResourceConfig{
						"deployment": {},
					},
				},
			},
		},
	}
	cm := childrenMap{
		"deployment": {"app": {"kind": "Deployment"}},
		"service":    {"svc": {"kind": "Service"}},
	}
	got := r.applyChildrenConfig(cm)
	assert.NotNil(t, got["deployment"])
	assert.Nil(t, got["service"])
}

func TestApplyChildrenConfig_RootExcludeStripsPath(t *testing.T) {
	r := &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Payload: &orktypes.RemotePayloadConfig{
				Children: &orktypes.RemotePayloadChildrenConfig{
					Exclude: []string{"metadata.managedFields"},
				},
			},
		},
	}
	cm := childrenMap{
		"deployment": {
			"app": {
				"kind": "Deployment",
				"metadata": map[string]interface{}{
					"name":          "app",
					"managedFields": []interface{}{"data"},
				},
			},
		},
	}
	got := r.applyChildrenConfig(cm)
	meta := got["deployment"]["app"]["metadata"].(map[string]interface{})
	assert.Nil(t, meta["managedFields"])
	assert.Equal(t, "app", meta["name"])
}

func TestApplyChildrenConfig_PerResourceExcludeOverridesRoot(t *testing.T) {
	r := &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Payload: &orktypes.RemotePayloadConfig{
				Children: &orktypes.RemotePayloadChildrenConfig{
					Exclude: []string{"metadata.managedFields"},
					Resources: map[string]*orktypes.RemotePayloadChildrenResourceConfig{
						"deployment": {Exclude: []string{"spec.template.metadata.annotations"}},
						"service":    nil, // uses root exclude
					},
				},
			},
		},
	}
	cm := childrenMap{
		"deployment": {
			"app": {
				"metadata": map[string]interface{}{"managedFields": []interface{}{"x"}},
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"metadata": map[string]interface{}{"annotations": map[string]interface{}{"k": "v"}},
					},
				},
			},
		},
		"service": {
			"svc": {
				"metadata": map[string]interface{}{"managedFields": []interface{}{"x"}, "name": "svc"},
			},
		},
	}
	got := r.applyChildrenConfig(cm)
	// deployment: per-resource exclude strips spec.template.metadata.annotations, NOT managedFields
	depMeta := got["deployment"]["app"]["metadata"].(map[string]interface{})
	assert.NotNil(t, depMeta["managedFields"])
	depTplMeta := got["deployment"]["app"]["spec"].(map[string]interface{})["template"].(map[string]interface{})["metadata"].(map[string]interface{})
	assert.Nil(t, depTplMeta["annotations"])
	// service: root exclude strips managedFields
	svcMeta := got["service"]["svc"]["metadata"].(map[string]interface{})
	assert.Nil(t, svcMeta["managedFields"])
}

func TestResolveArgs_NilPrepared_FallsBack(t *testing.T) {
	raw := map[string]interface{}{"env": "{{ .spec.env }}"}
	r := &Reconciler{
		decl: &orktypes.RemoteReconcilerDeclaration{
			Endpoint: "http://svc/reconcile",
			Args:     raw,
		},
	}
	got := r.resolveArgs(domain.Request{})
	assert.Equal(t, raw, got)
}
