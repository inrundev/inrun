// pkg/types/types_runtime_config.go
package types

// RuntimeConfig governs the operatorBox as a long-lived entity, not a single reconcile cycle.
// Covers autoscaling the worker pool, rollback on error, finalizer lifecycle,
// namespace guards, and deletion protection.
type RuntimeConfig struct {
	// Finalizers is the per-CRD finalizer list. Falls back to the Katalog-level finalizer.
	Finalizers []string `yaml:"finalizers,omitempty" json:"finalizers,omitempty" validate:"omitempty"`

	// RemoveFinalizers strips all Orkestra finalizers from this CRD's CRs. Testing only.
	RemoveFinalizers bool `yaml:"removeFinalizers,omitempty" json:"removeFinalizers,omitempty"`

	// DeletionProtection overrides the global deletion protection policy for this CRD.
	DeletionProtection *DeletionProtectionOverride `yaml:"deletionProtection,omitempty" json:"deletionProtection,omitempty"`

	// RestrictedNamespaces blocks reconciliation for CRs in the named namespaces.
	RestrictedNamespaces RestrictedNamespaces `yaml:"restrictedNamespaces,omitempty" json:"restrictedNamespaces,omitempty"`

	// AllowedNamespaces restricts reconciliation to CRs in the named namespaces only.
	AllowedNamespaces AllowedNamespaces `yaml:"allowedNamespaces,omitempty" json:"allowedNamespaces,omitempty"`

	// IgnoreStatusPatch disables the runtime's automatic status patch for this CRD.
	IgnoreStatusPatch bool `yaml:"ignoreStatusPatch,omitempty" json:"ignoreStatusPatch,omitempty"`

	// IgnoreObservedGeneration disables generation-based reconcile skipping for this CRD.
	IgnoreObservedGeneration bool `yaml:"ignoreObservedGeneration,omitempty" json:"ignoreObservedGeneration,omitempty"`

	// Autoscale declares runtime worker/queue/resync overrides driven by conditions.
	Autoscale *AutoscaleSpec `yaml:"autoscale,omitempty" json:"autoscale,omitempty"`

	// Rollback declares failure-recovery behavior.
	Rollback *RollbackBlock `yaml:"rollback,omitempty" json:"rollback,omitempty"`

	// RollBackOnError enables zero-config rollback on 3 consecutive failures.
	RollBackOnError bool `yaml:"rollBackOnError,omitempty" json:"rollBackOnError,omitempty"`
}

func (r *RuntimeConfig) Empty() bool { return r == nil }
