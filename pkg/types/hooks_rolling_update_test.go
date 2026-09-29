package types_test

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func crdWithDeploymentOnCreate(deps ...orktypes.DeploymentTemplateSource) orktypes.CRDEntry {
	return orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnCreate: &orktypes.HookTemplates{
					Deployments: deps,
				},
			},
		},
	}
}

func TestCollectRollingUpdateProfileEntries_Empty(t *testing.T) {
	c := orktypes.CRDEntry{}
	assert.Empty(t, c.CollectRollingUpdateProfileEntries())
}

func TestCollectRollingUpdateProfileEntries_NoRollingUpdate(t *testing.T) {
	c := crdWithDeploymentOnCreate(orktypes.DeploymentTemplateSource{Name: "app"})
	assert.Empty(t, c.CollectRollingUpdateProfileEntries())
}

func TestCollectRollingUpdateProfileEntries_RollingUpdateNoProfile(t *testing.T) {
	c := crdWithDeploymentOnCreate(orktypes.DeploymentTemplateSource{
		Name:          "app",
		RollingUpdate: &orktypes.RollingUpdateBehavior{},
	})
	assert.Empty(t, c.CollectRollingUpdateProfileEntries())
}

func TestCollectRollingUpdateProfileEntries_ProfileReturned(t *testing.T) {
	c := crdWithDeploymentOnCreate(orktypes.DeploymentTemplateSource{
		Name:          "app",
		RollingUpdate: &orktypes.RollingUpdateBehavior{Profile: "safe"},
	})
	entries := c.CollectRollingUpdateProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "onCreate", entries[0].Phase)
	assert.Equal(t, "app", entries[0].ResourceName)
	assert.Equal(t, "safe", entries[0].Profile)
	assert.False(t, entries[0].Mixed)
}

func TestCollectRollingUpdateProfileEntries_Mixed_MaxSurge(t *testing.T) {
	c := crdWithDeploymentOnCreate(orktypes.DeploymentTemplateSource{
		Name: "app",
		RollingUpdate: &orktypes.RollingUpdateBehavior{
			Profile:  "fast",
			MaxSurge: "2",
		},
	})
	entries := c.CollectRollingUpdateProfileEntries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Mixed)
}

func TestCollectRollingUpdateProfileEntries_Mixed_MaxUnavailable(t *testing.T) {
	c := crdWithDeploymentOnCreate(orktypes.DeploymentTemplateSource{
		Name: "app",
		RollingUpdate: &orktypes.RollingUpdateBehavior{
			Profile:        "blue-green",
			MaxUnavailable: "0",
		},
	})
	entries := c.CollectRollingUpdateProfileEntries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Mixed)
}

func TestCollectRollingUpdateProfileEntries_TemplateExpr(t *testing.T) {
	c := crdWithDeploymentOnCreate(orktypes.DeploymentTemplateSource{
		RollingUpdate: &orktypes.RollingUpdateBehavior{Profile: "{{ .Spec.DeployProfile }}"},
	})
	entries := c.CollectRollingUpdateProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "{{ .Spec.DeployProfile }}", entries[0].Profile)
}

func TestCollectRollingUpdateProfileEntries_StatefulSet(t *testing.T) {
	c := orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnCreate: &orktypes.HookTemplates{
					StatefulSets: []orktypes.StatefulSetTemplateSource{
						{Name: "db", RollingUpdate: &orktypes.RollingUpdateBehavior{Profile: "safe"}},
					},
				},
			},
		},
	}
	entries := c.CollectRollingUpdateProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "safe", entries[0].Profile)
	assert.Equal(t, "db", entries[0].ResourceName)
}

func TestCollectRollingUpdateProfileEntries_OnReconcile(t *testing.T) {
	c := orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnReconcile: &orktypes.HookTemplates{
					Deployments: []orktypes.DeploymentTemplateSource{
						{Name: "app", RollingUpdate: &orktypes.RollingUpdateBehavior{Profile: "fast"}},
					},
				},
			},
		},
	}
	entries := c.CollectRollingUpdateProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "onReconcile", entries[0].Phase)
	assert.Equal(t, "fast", entries[0].Profile)
}
