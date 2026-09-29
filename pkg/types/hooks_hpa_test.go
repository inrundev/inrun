package types_test

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func crdWithHPAOnCreate(hpas ...orktypes.HPATemplateSource) orktypes.CRDEntry {
	return orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnCreate: &orktypes.HookTemplates{
					HorizontalPodAutoscalers: hpas,
				},
			},
		},
	}
}

func TestCollectHPAProfileEntries_Empty(t *testing.T) {
	c := orktypes.CRDEntry{}
	assert.Empty(t, c.CollectHPAProfileEntries())
}

func TestCollectHPAProfileEntries_NoBehavior(t *testing.T) {
	c := crdWithHPAOnCreate(orktypes.HPATemplateSource{Name: "hpa"})
	assert.Empty(t, c.CollectHPAProfileEntries())
}

func TestCollectHPAProfileEntries_BehaviorNoProfile(t *testing.T) {
	c := crdWithHPAOnCreate(orktypes.HPATemplateSource{
		Name:     "hpa",
		Behavior: &orktypes.HPABehavior{},
	})
	assert.Empty(t, c.CollectHPAProfileEntries())
}

func TestCollectHPAProfileEntries_ProfileReturned(t *testing.T) {
	c := crdWithHPAOnCreate(orktypes.HPATemplateSource{
		Name:     "my-hpa",
		Behavior: &orktypes.HPABehavior{Profile: "web"},
	})
	entries := c.CollectHPAProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "onCreate", entries[0].Phase)
	assert.Equal(t, "my-hpa", entries[0].ResourceName)
	assert.Equal(t, "web", entries[0].Profile)
	assert.False(t, entries[0].Mixed)
}

func TestCollectHPAProfileEntries_Mixed_ScaleUp(t *testing.T) {
	c := crdWithHPAOnCreate(orktypes.HPATemplateSource{
		Name: "hpa",
		Behavior: &orktypes.HPABehavior{
			Profile: "batch",
			ScaleUp: &orktypes.HPAScalingRules{StabilizationWindowSeconds: 30},
		},
	})
	entries := c.CollectHPAProfileEntries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Mixed)
}

func TestCollectHPAProfileEntries_Mixed_ScaleDown(t *testing.T) {
	c := crdWithHPAOnCreate(orktypes.HPATemplateSource{
		Name: "hpa",
		Behavior: &orktypes.HPABehavior{
			Profile:   "cost-optimized",
			ScaleDown: &orktypes.HPAScalingRules{StabilizationWindowSeconds: 300},
		},
	})
	entries := c.CollectHPAProfileEntries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Mixed)
}

func TestCollectHPAProfileEntries_TemplateExpr(t *testing.T) {
	c := crdWithHPAOnCreate(orktypes.HPATemplateSource{
		Behavior: &orktypes.HPABehavior{Profile: "{{ .Spec.ScaleProfile }}"},
	})
	entries := c.CollectHPAProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "{{ .Spec.ScaleProfile }}", entries[0].Profile)
}

func TestCollectHPAProfileEntries_OnReconcile(t *testing.T) {
	c := orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnReconcile: &orktypes.HookTemplates{
					HorizontalPodAutoscalers: []orktypes.HPATemplateSource{
						{Name: "hpa", Behavior: &orktypes.HPABehavior{Profile: "api"}},
					},
				},
			},
		},
	}
	entries := c.CollectHPAProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "onReconcile", entries[0].Phase)
	assert.Equal(t, "api", entries[0].Profile)
}
