package validate

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func katalogWithLabels(crdName string, labels orktypes.Labels) *executor {
	return newKatalogExec(map[string]orktypes.CRDEntry{
		crdName: {Labels: labels},
	})
}

func katalogWithAnnotations(crdName string, annotations orktypes.Labels) *executor {
	return newKatalogExec(map[string]orktypes.CRDEntry{
		crdName: {Annotations: annotations},
	})
}

// ── Labels ────────────────────────────────────────────────────────────────────

func TestValidateCRDEntryLabels_NoLabels(t *testing.T) {
	k := katalogWithLabels("myapp", nil)
	assert.NoError(t, k.validateCRDEntryLabels())
}

func TestValidateCRDEntryLabels_ValidStaticKey(t *testing.T) {
	k := katalogWithLabels("myapp", orktypes.Labels{
		"app.kubernetes.io/name": "{{ .metadata.name }}",
	})
	assert.NoError(t, k.validateCRDEntryLabels())
}

func TestValidateCRDEntryLabels_ValidPlainValue(t *testing.T) {
	k := katalogWithLabels("myapp", orktypes.Labels{
		"env": "production",
	})
	assert.NoError(t, k.validateCRDEntryLabels())
}

func TestValidateCRDEntryLabels_TemplateKey_Rejected(t *testing.T) {
	k := katalogWithLabels("myapp", orktypes.Labels{
		"{{ .metadata.name }}": "value",
	})
	err := k.validateCRDEntryLabels()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a static label key")
	assert.Contains(t, err.Error(), "myapp")
}

func TestValidateCRDEntryLabels_InvalidKey_Rejected(t *testing.T) {
	k := katalogWithLabels("myapp", orktypes.Labels{
		"invalid key!": "value",
	})
	err := k.validateCRDEntryLabels()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid Kubernetes label key")
	assert.Contains(t, err.Error(), "myapp")
}

func TestValidateCRDEntryLabels_InvalidValueTemplate_Rejected(t *testing.T) {
	k := katalogWithLabels("myapp", orktypes.Labels{
		"tier": "{{ .metadata.name",
	})
	err := k.validateCRDEntryLabels()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template")
}

func TestValidateCRDEntryLabels_InvalidValue_Rejected(t *testing.T) {
	k := katalogWithLabels("myapp", orktypes.Labels{
		"tier": "value/with/slashes",
	})
	err := k.validateCRDEntryLabels()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a valid Kubernetes label value")
}

func TestValidateCRDEntryLabels_MultipleKeys_OneInvalid(t *testing.T) {
	k := katalogWithLabels("myapp", orktypes.Labels{
		"app":          "{{ .metadata.name }}",
		"invalid key!": "value",
	})
	err := k.validateCRDEntryLabels()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid Kubernetes label key")
}

// ── Annotations ───────────────────────────────────────────────────────────────

func TestValidateCRDEntryAnnotations_NoAnnotations(t *testing.T) {
	k := katalogWithAnnotations("myapp", nil)
	assert.NoError(t, k.validateCRDEntryAnnotations())
}

func TestValidateCRDEntryAnnotations_ValidStaticKey(t *testing.T) {
	k := katalogWithAnnotations("myapp", orktypes.Labels{
		"example.io/owner": "{{ .spec.owner }}",
	})
	assert.NoError(t, k.validateCRDEntryAnnotations())
}

func TestValidateCRDEntryAnnotations_ValidPlainValue(t *testing.T) {
	k := katalogWithAnnotations("myapp", orktypes.Labels{
		"example.io/region": "us-east-1",
	})
	assert.NoError(t, k.validateCRDEntryAnnotations())
}

func TestValidateCRDEntryAnnotations_TemplateKey_Rejected(t *testing.T) {
	k := katalogWithAnnotations("myapp", orktypes.Labels{
		"{{ .metadata.name }}": "value",
	})
	err := k.validateCRDEntryAnnotations()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a static annotation key")
	assert.Contains(t, err.Error(), "myapp")
}

func TestValidateCRDEntryAnnotations_InvalidKey_Rejected(t *testing.T) {
	k := katalogWithAnnotations("myapp", orktypes.Labels{
		"invalid annotation!": "value",
	})
	err := k.validateCRDEntryAnnotations()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid Kubernetes annotation key")
	assert.Contains(t, err.Error(), "myapp")
}

func TestValidateCRDEntryAnnotations_InvalidValueTemplate_Rejected(t *testing.T) {
	k := katalogWithAnnotations("myapp", orktypes.Labels{
		"example.io/desc": "{{ .spec.name",
	})
	err := k.validateCRDEntryAnnotations()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template")
}

// ── Combined (validateCRDEntryMetadata) ──────────────────────────────────────

func TestValidateCRDEntryMetadata_BothValid(t *testing.T) {
	k := newKatalogExec(map[string]orktypes.CRDEntry{
		"myapp": {
			Labels:      orktypes.Labels{"app": "{{ .metadata.name }}"},
			Annotations: orktypes.Labels{"example.io/owner": "team-a"},
		},
	})
	assert.NoError(t, k.validateCRDEntryMetadata())
}

func TestValidateCRDEntryMetadata_InvalidLabel_Short_Circuits(t *testing.T) {
	k := newKatalogExec(map[string]orktypes.CRDEntry{
		"myapp": {
			Labels:      orktypes.Labels{"invalid key!": "value"},
			Annotations: orktypes.Labels{"example.io/ok": "value"},
		},
	})
	err := k.validateCRDEntryMetadata()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid Kubernetes label key")
}

func TestValidateCRDEntryMetadata_InvalidAnnotation_Caught(t *testing.T) {
	k := newKatalogExec(map[string]orktypes.CRDEntry{
		"myapp": {
			Labels:      orktypes.Labels{"app": "myapp"},
			Annotations: orktypes.Labels{"bad annotation!": "value"},
		},
	})
	err := k.validateCRDEntryMetadata()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid Kubernetes annotation key")
}
