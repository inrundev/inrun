// pkg/types/rollback.go
//
// Rollback — declarative failure recovery for operatorboxes.
//
// When a reconcile fails a configurable number of times (or within a time
// window), Orkestra enters a rollback phase. In this phase it applies the
// previous desired state — captured before the failing spec change was applied
// — and blocks normal reconciliation until the operator fixes the spec.
//
// Rollback is not "undo" in the transactional sense. It is "re-apply the last
// known good declaration." The existing Update functions handle idempotent
// re-application — no new primitives are needed.
//
// YAML shape (inside operatorBox:):
//
//	operatorBox:
//	  rollback:
//	    trigger:
//	      consecutiveFailures: 3    # OR
//	      withinDuration: 5m        # triggers if N failures occur within this window
//	    onRollback:
//	      deployments:
//	        - name: "{{ .previous.metadata.name }}"
//	          image: "{{ .previous.spec.image }}"
//	          replicas: "{{ .previous.spec.replicas }}"
//	          reconcile: true
//	      configMaps:
//	        - name: "{{ .previous.metadata.name }}-config"
//	          reconcile: true
//
// The .previous.* context is hydrated from the annotation:
//
//	orkestra.orkspace.io/previous-spec
//
// which is written atomically before each spec change is applied.
//
// The rollback phase exits only when the spec changes (new generation).
// Until then, normal onCreate/onReconcile is blocked.
package types

import "time"

// RollbackBlock declares the rollback behavior for one operatorbox.
// Declared inside OperatorBoxConfig as Rollback *RollbackBlock.
type RollbackBlock struct {
	// Trigger declares when rollback activates.
	// Both fields are optional — set one or both.
	// When both are set: triggers when ConsecutiveFailures failures
	// occur AND those failures all happened within WithinDuration.
	Trigger RollbackTrigger `yaml:"trigger" json:"trigger"`

	// OnRollback declares the resource groups to apply when rollback is active.
	// Uses the same grammar as OnReconcile — deployments, services, configMaps, etc.
	// Templates may reference .previous.spec.*, .previous.metadata.*, etc.
	// which are hydrated from the previous-spec annotation.
	OnRollback *HookTemplates `yaml:"onRollback,omitempty" json:"onRollback,omitempty"`
}

// RollbackTrigger declares the conditions that activate rollback.
type RollbackTrigger struct {
	// ConsecutiveFailures is the number of consecutive reconcile failures
	// that triggers rollback. Default: 3.
	ConsecutiveFailures int `yaml:"consecutiveFailures,omitempty" json:"consecutiveFailures,omitempty"`

	// WithinDuration is the time window within which ConsecutiveFailures
	// must occur for rollback to trigger.
	// When omitted: any ConsecutiveFailures consecutive failures trigger rollback,
	// regardless of how long ago the first failure occurred.
	// When set: only triggers if all failures occurred within this window.
	// Example: 3 failures within 5m = trigger; 3 failures over 2 hours = no trigger.
	WithinDuration *Duration `yaml:"withinDuration,omitempty" json:"withinDuration,omitempty"`
}

// RollbackPhase represents the reconciler's rollback state machine.
// Stored in status.phase — the reconciler reads this on every cycle.
type RollbackPhase string

const (
	// RollbackPhaseNone — normal reconciliation, no rollback active.
	RollbackPhaseNone RollbackPhase = ""

	// RollbackPhaseActive — rollback is active. Normal reconciliation blocked.
	// The previous spec is being applied. Exits when spec generation changes.
	RollbackPhaseActive RollbackPhase = "RolledBack"
)

// PreviousSpecAnnotation is the annotation key where the previous spec
// is stored before a spec change is applied.
// Value is a base64-encoded, gzip-compressed JSON of the CR spec.
const PreviousSpecAnnotation = "orkestra.orkspace.io/previous-spec"

// RollbackGenerationAnnotation records the generation at which rollback
// was last triggered. Used to detect spec changes that should exit rollback.
const RollbackGenerationAnnotation = "orkestra.orkspace.io/rollback-at-generation"

// EffectiveConsecutiveFailures returns the consecutive failure threshold,
// applying the default of 3 when not declared.
func (t *RollbackTrigger) EffectiveConsecutiveFailures() int {
	if t.ConsecutiveFailures <= 0 {
		return 3
	}
	return t.ConsecutiveFailures
}

// ── DerivedRollback ───────────────────────────────────────────────────────────

// DerivedRollback returns the effective RollbackBlock for this operatorbox,
// merging rollBackOnError: true shorthand with any explicit rollback: declaration.
//
// Resolution order:
//  1. Neither rollBackOnError nor rollback set → nil (no rollback)
//  2. rollBackOnError: false, rollback set → explicit block returned as-is
//  3. rollBackOnError: true → synthetic block derived from reconcile:true resources;
//     explicit rollback.trigger and rollback.onRollback take precedence when set
func (c *OperatorBoxConfig) DerivedRollback() *RollbackBlock {
	rollback := c.EffectiveRollback()
	rollBackOnError := c.EffectiveRollBackOnError()
	if !rollBackOnError && rollback == nil {
		return nil
	}
	if !rollBackOnError {
		return rollback
	}

	block := &RollbackBlock{}

	// Merge trigger from explicit block if declared
	if rollback != nil {
		block.Trigger = rollback.Trigger
	}

	// Explicit onRollback takes precedence over derived templates
	if rollback != nil && rollback.OnRollback != nil {
		block.OnRollback = rollback.OnRollback
		return block
	}

	block.OnRollback = deriveRollbackTemplates(c.EffectiveOnCreate(), c.EffectiveOnReconcile())
	return block
}

// deriveRollbackTemplates collects all resource declarations with reconcile: true
// from onCreate and onReconcile into a synthetic HookTemplates for rollback.
// Resources with once: true (generated secrets) are excluded.
//
// The returned templates are intended to be run with a resolver where .spec.*
// has been substituted with the previous spec — so the same template expressions
// ({{ .spec.image }}) work for both normal reconcile and rollback.
func deriveRollbackTemplates(onCreate, onReconcile *HookTemplates) *HookTemplates {
	derived := &HookTemplates{}

	collect := func(src *HookTemplates) {
		if src == nil {
			return
		}
		for _, d := range src.Deployments {
			if d.Reconcile {
				derived.Deployments = append(derived.Deployments, d)
			}
		}
		for _, s := range src.Services {
			if s.Reconcile {
				derived.Services = append(derived.Services, s)
			}
		}
		for _, cm := range src.ConfigMaps {
			if cm.Reconcile {
				derived.ConfigMaps = append(derived.ConfigMaps, cm)
			}
		}
		for _, ss := range src.StatefulSets {
			if ss.Reconcile {
				derived.StatefulSets = append(derived.StatefulSets, ss)
			}
		}
		for _, rs := range src.ReplicaSets {
			if rs.Reconcile {
				derived.ReplicaSets = append(derived.ReplicaSets, rs)
			}
		}
		for _, ing := range src.Ingresses {
			if ing.Reconcile {
				derived.Ingresses = append(derived.Ingresses, ing)
			}
		}
		for _, hpa := range src.HorizontalPodAutoscalers {
			if hpa.Reconcile {
				derived.HorizontalPodAutoscalers = append(derived.HorizontalPodAutoscalers, hpa)
			}
		}
		for _, pdb := range src.PodDisruptionBudgets {
			if pdb.Reconcile {
				derived.PodDisruptionBudgets = append(derived.PodDisruptionBudgets, pdb)
			}
		}
		// Secrets: only include those without once:true
		for _, sec := range src.Secrets {
			if sec.Reconcile && !sec.Once {
				derived.Secrets = append(derived.Secrets, sec)
			}
		}
	}

	collect(onCreate)
	collect(onReconcile)
	return derived
}

// ShouldTrigger returns true when the failure history meets the rollback criteria.
//
// failures is the list of failure timestamps in reverse chronological order
// (most recent first). The function checks whether the required number of
// consecutive failures have occurred within the configured window.
func (t *RollbackTrigger) ShouldTrigger(failureTimes []time.Time) bool {
	required := t.EffectiveConsecutiveFailures()
	if len(failureTimes) < required {
		return false
	}

	// No window constraint — any N consecutive failures trigger
	if t.WithinDuration == nil || t.WithinDuration.Duration == 0 {
		return len(failureTimes) >= required
	}

	// Window constraint — all N failures must be within the window
	window := t.WithinDuration.Duration
	cutoff := time.Now().Add(-window)
	count := 0
	for _, ts := range failureTimes {
		if ts.Before(cutoff) {
			break // failures are newest-first; once we're past the window, stop
		}
		count++
		if count >= required {
			return true
		}
	}
	return false
}
