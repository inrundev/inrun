// Tests for RollbackTrigger and OperatorBoxConfig rollback helpers (rollback.go).
package types_test

import (
	"testing"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── RollbackTrigger.EffectiveConsecutiveFailures ──────────────────────────────

func TestEffectiveConsecutiveFailures_Default(t *testing.T) {
	var tr orktypes.RollbackTrigger
	assert.Equal(t, 3, tr.EffectiveConsecutiveFailures())
}

func TestEffectiveConsecutiveFailures_ZeroIsDefault(t *testing.T) {
	tr := orktypes.RollbackTrigger{ConsecutiveFailures: 0}
	assert.Equal(t, 3, tr.EffectiveConsecutiveFailures())
}

func TestEffectiveConsecutiveFailures_NegativeIsDefault(t *testing.T) {
	tr := orktypes.RollbackTrigger{ConsecutiveFailures: -1}
	assert.Equal(t, 3, tr.EffectiveConsecutiveFailures())
}

func TestEffectiveConsecutiveFailures_Custom(t *testing.T) {
	tr := orktypes.RollbackTrigger{ConsecutiveFailures: 5}
	assert.Equal(t, 5, tr.EffectiveConsecutiveFailures())
}

// ── RollbackTrigger.ShouldTrigger ────────────────────────────────────────────

func TestShouldTrigger_NotEnoughFailures(t *testing.T) {
	tr := orktypes.RollbackTrigger{ConsecutiveFailures: 3}
	failures := []time.Time{time.Now(), time.Now()}
	assert.False(t, tr.ShouldTrigger(failures))
}

func TestShouldTrigger_ExactThresholdNoWindow(t *testing.T) {
	tr := orktypes.RollbackTrigger{ConsecutiveFailures: 3}
	failures := []time.Time{time.Now(), time.Now(), time.Now()}
	assert.True(t, tr.ShouldTrigger(failures))
}

func TestShouldTrigger_ExceedsThresholdNoWindow(t *testing.T) {
	tr := orktypes.RollbackTrigger{ConsecutiveFailures: 2}
	failures := []time.Time{time.Now(), time.Now(), time.Now()}
	assert.True(t, tr.ShouldTrigger(failures))
}

func TestShouldTrigger_EmptyFailures(t *testing.T) {
	tr := orktypes.RollbackTrigger{ConsecutiveFailures: 1}
	assert.False(t, tr.ShouldTrigger(nil))
}

func TestShouldTrigger_WithWindowAllWithin(t *testing.T) {
	d := orktypes.Duration{Duration: 10 * time.Minute}
	tr := orktypes.RollbackTrigger{
		ConsecutiveFailures: 3,
		WithinDuration:      &d,
	}
	now := time.Now()
	failures := []time.Time{
		now.Add(-1 * time.Minute),
		now.Add(-2 * time.Minute),
		now.Add(-3 * time.Minute),
	}
	assert.True(t, tr.ShouldTrigger(failures))
}

func TestShouldTrigger_WithWindowSomeOutside(t *testing.T) {
	d := orktypes.Duration{Duration: 5 * time.Minute}
	tr := orktypes.RollbackTrigger{
		ConsecutiveFailures: 3,
		WithinDuration:      &d,
	}
	now := time.Now()
	// Third failure is outside the 5m window
	failures := []time.Time{
		now.Add(-1 * time.Minute),
		now.Add(-2 * time.Minute),
		now.Add(-10 * time.Minute), // outside window
	}
	assert.False(t, tr.ShouldTrigger(failures))
}

// ── OperatorBoxConfig.DerivedRollback ────────────────────────────────────────

func TestDerivedRollback_NeitherSet(t *testing.T) {
	c := orktypes.OperatorBoxConfig{}
	assert.Nil(t, c.DerivedRollback())
}

func TestDerivedRollback_OnlyExplicitBlock(t *testing.T) {
	rollback := &orktypes.RollbackBlock{}
	c := orktypes.OperatorBoxConfig{
		Runtime: &orktypes.RuntimeConfig{
			Rollback: rollback,
		},
	}
	result := c.DerivedRollback()
	assert.NotNil(t, result)
	assert.Same(t, rollback, result)
}

func TestDerivedRollback_ShorthandNoOnCreate(t *testing.T) {
	c := orktypes.OperatorBoxConfig{Runtime: &orktypes.RuntimeConfig{RollBackOnError: true}}
	result := c.DerivedRollback()
	assert.NotNil(t, result)
	// No reconcile:true resources declared → empty OnRollback templates
	assert.NotNil(t, result.OnRollback)
}

func TestDerivedRollback_ShorthandWithReconcileDeployment(t *testing.T) {
	c := orktypes.OperatorBoxConfig{
		Runtime: &orktypes.RuntimeConfig{RollBackOnError: true},
		Reconcile: &orktypes.ReconcileConfig{
			OnCreate: &orktypes.HookTemplates{
				Deployments: []orktypes.DeploymentTemplateSource{
					{Reconcile: true},
					{Reconcile: false}, // excluded
				},
			},
		},
	}
	result := c.DerivedRollback()
	require.NotNil(t, result)
	require.NotNil(t, result.OnRollback)
	assert.Len(t, result.OnRollback.Deployments, 1, "only reconcile:true deployments are included")
}

func TestDerivedRollback_ShorthandExplicitOnRollbackTakesPrecedence(t *testing.T) {
	explicit := &orktypes.HookTemplates{
		Services: []orktypes.ServiceTemplateSource{{Name: "override"}},
	}
	c := orktypes.OperatorBoxConfig{
		Runtime: &orktypes.RuntimeConfig{
			RollBackOnError: true,
			Rollback:        &orktypes.RollbackBlock{OnRollback: explicit},
		},
		Reconcile: &orktypes.ReconcileConfig{
			OnCreate: &orktypes.HookTemplates{
				Deployments: []orktypes.DeploymentTemplateSource{{Reconcile: true}},
			},
		},
	}
	result := c.DerivedRollback()
	require.NotNil(t, result)
	// Explicit OnRollback wins over derived templates
	assert.Same(t, explicit, result.OnRollback)
	assert.Empty(t, result.OnRollback.Deployments)
}
