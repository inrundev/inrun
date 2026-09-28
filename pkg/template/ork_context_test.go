package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestOrkContext_InjectedIntoResolver(t *testing.T) {
	orkCtx := NewOrkContext("orkestra-system", "v0.7.18")
	ctx := ContextWithOrkContext(context.Background(), orkCtx)

	obj := &unstructured.Unstructured{}
	obj.SetName("my-app")
	obj.SetNamespace("default")

	r, err := NewResolver(ctx, obj)
	require.NoError(t, err)

	got, ok := r.data["ork"].(map[string]interface{})
	require.True(t, ok, "expected .ork to be a map")

	assert.Equal(t, "orkestra-system", got["namespace"])
	assert.Equal(t, "v0.7.18", got["version"])
	assert.Equal(t, false, got["inCluster"]) // not running inside a pod in tests
}

func TestOrkContext_ZeroValueWhenAbsent(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetName("my-app")
	obj.SetNamespace("default")

	r, err := NewResolver(context.Background(), obj)
	require.NoError(t, err)

	got, ok := r.data["ork"].(map[string]interface{})
	require.True(t, ok, "expected .ork to be a map even without context")
	assert.Equal(t, "", got["namespace"])
	assert.Equal(t, "", got["version"])
}

func TestOrkContext_TemplateExpression(t *testing.T) {
	orkCtx := NewOrkContext("orkestra-system", "v0.7.18")
	ctx := ContextWithOrkContext(context.Background(), orkCtx)

	obj := &unstructured.Unstructured{}
	obj.SetName("my-app")
	obj.SetNamespace("default")

	r, err := NewResolver(ctx, obj)
	require.NoError(t, err)

	result, err := r.Resolve("http://reconciler.{{ .ork.namespace }}.svc.cluster.local/reconcile")
	require.NoError(t, err)
	assert.Equal(t, "http://reconciler.orkestra-system.svc.cluster.local/reconcile", result)
}
