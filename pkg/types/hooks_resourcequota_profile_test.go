package types_test

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func crdWithResourceQuotaOnCreate(rqs ...orktypes.ResourceQuotaTemplateSource) orktypes.CRDEntry {
	return orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnCreate: &orktypes.HookTemplates{
					ResourceQuotas: rqs,
				},
			},
		},
	}
}

func TestCollectResourceQuotaProfileEntries_Empty(t *testing.T) {
	c := orktypes.CRDEntry{}
	assert.Empty(t, c.CollectResourceQuotaProfileEntries())
}

func TestCollectResourceQuotaProfileEntries_NoProfile(t *testing.T) {
	c := crdWithResourceQuotaOnCreate(orktypes.ResourceQuotaTemplateSource{Name: "quota"})
	assert.Empty(t, c.CollectResourceQuotaProfileEntries())
}

func TestCollectResourceQuotaProfileEntries_ProfileReturned(t *testing.T) {
	c := crdWithResourceQuotaOnCreate(orktypes.ResourceQuotaTemplateSource{
		Name:    "default-quota",
		Profile: "small",
	})
	entries := c.CollectResourceQuotaProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "onCreate", entries[0].Phase)
	assert.Equal(t, "default-quota", entries[0].ResourceName)
	assert.Equal(t, "small", entries[0].Profile)
	assert.False(t, entries[0].Mixed)
}

func TestCollectResourceQuotaProfileEntries_Mixed(t *testing.T) {
	c := crdWithResourceQuotaOnCreate(orktypes.ResourceQuotaTemplateSource{
		Name:    "quota",
		Profile: "medium",
		Hard:    map[string]string{"pods": "5"},
	})
	entries := c.CollectResourceQuotaProfileEntries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Mixed)
}

func TestCollectResourceQuotaProfileEntries_TemplateExpr(t *testing.T) {
	c := crdWithResourceQuotaOnCreate(orktypes.ResourceQuotaTemplateSource{
		Profile: "{{ .Spec.QuotaProfile }}",
	})
	entries := c.CollectResourceQuotaProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "{{ .Spec.QuotaProfile }}", entries[0].Profile)
}

func TestCollectResourceQuotaProfileEntries_MultipleProfiles(t *testing.T) {
	c := orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnCreate: &orktypes.HookTemplates{
					ResourceQuotas: []orktypes.ResourceQuotaTemplateSource{
						{Name: "rq-small", Profile: "small"},
						{Name: "rq-no-profile"},
						{Name: "rq-large", Profile: "large"},
					},
				},
			},
		},
	}
	entries := c.CollectResourceQuotaProfileEntries()
	require.Len(t, entries, 2)
	assert.Equal(t, "small", entries[0].Profile)
	assert.Equal(t, "large", entries[1].Profile)
}

func TestCollectResourceQuotaProfileEntries_OnReconcile(t *testing.T) {
	c := orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnReconcile: &orktypes.HookTemplates{
					ResourceQuotas: []orktypes.ResourceQuotaTemplateSource{
						{Name: "quota", Profile: "xlarge"},
					},
				},
			},
		},
	}
	entries := c.CollectResourceQuotaProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "onReconcile", entries[0].Phase)
	assert.Equal(t, "xlarge", entries[0].Profile)
}
