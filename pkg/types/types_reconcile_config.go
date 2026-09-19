// pkg/types/types_reconcile_config.go
package types

import "github.com/orkspace/orkestra/domain"

// ReconcileConfig declares what runs and how. Default (omitted or default: true) uses the
// GenericReconciler driven by onCreate/onReconcile/onDelete templates. Set default: false
// and declare constructor: to bring a typed Go reconciler. Workers, resync, queue, and
// requeue tune execution regardless of which reconciler runs.
type ReconcileConfig struct {
	// --- Implementation identity ---

	// Default: true → GenericReconciler (default when omitted).
	// false → custom reconciler via Constructor.
	Default *bool `yaml:"default,omitempty" json:"default,omitempty" validate:"omitempty"`

	// Hooks declares a Go hook function. Signature: func() domain.AnyReconcileHooks.
	Hooks *HookDeclaration `yaml:"hooks,omitempty" json:"hooks,omitempty" validate:"omitempty"`

	// Constructor declares a custom reconciler constructor. Required when Default: false.
	ConstructorDecl *ConstructorDeclaration `yaml:"constructor,omitempty" json:"constructor,omitempty" validate:"omitempty"`

	// --- Execution parameters ---

	// Profile is a named tuning preset. Built-ins: high-throughput, conservative, development.
	// Inline Workers/Resync/Queue override the profile.
	Profile string `yaml:"profile,omitempty" json:"profile,omitempty"`

	// Workers is the number of concurrent reconcile goroutines. 0 → DEFAULT_WORKERS.
	Workers int `yaml:"workers,omitempty" json:"workers,omitempty" validate:"omitempty,gte=1,lte=50"`

	// Resync is the full re-list interval for the informer cache. 0 → DEFAULT_RESYNC.
	Resync Duration `yaml:"resync,omitempty" json:"resync,omitempty"`

	Queue   Queue          `yaml:"queue,omitempty" json:"queue,omitempty"`
	Requeue *RequeueConfig `yaml:"requeue,omitempty" json:"requeue,omitempty"`

	// --- Declarative lifecycle ---

	// Normalize normalizes declared spec fields before template rendering.
	Normalize *NormalizeConfig `yaml:"normalize,omitempty" json:"normalize,omitempty"`

	// Imports declares Motif imports merged into onReconcile at load time.
	Imports []MotifImport `yaml:"imports,omitempty" json:"imports,omitempty"`

	// ForceConflict sets Force: true on server-side apply for child resources.
	ForceConflict *bool `yaml:"forceConflict,omitempty" json:"forceConflict,omitempty"`

	// RawProviders is the raw providers: map. Converted to ProviderBlocks after unmarshal.
	RawProviders map[string][]map[string]interface{} `yaml:"providers,omitempty" json:"providers,omitempty"`

	OnCreate    *HookTemplates `yaml:"onCreate,omitempty" json:"onCreate,omitempty" validate:"omitempty"`
	OnReconcile *HookTemplates `yaml:"onReconcile,omitempty" json:"onReconcile,omitempty" validate:"omitempty"`
	OnDelete    *HookTemplates `yaml:"onDelete,omitempty" json:"onDelete,omitempty" validate:"omitempty"`

	// --- At runtime mapping ---

	HookFactory    func() domain.AnyReconcileHooks `yaml:"-" json:"-"`
	Constructor    NewReconcilerFunc               `yaml:"-" json:"-"`
	ProviderBlocks []ProviderBlock                 `yaml:"-" json:"-"`

	// Include is a path to a YAML file whose reconcile: block is merged under this config.
	// Inline fields take precedence. Cleared after expansion.
	Include string `yaml:"include,omitempty" json:"include,omitempty"`
}

func (r *ReconcileConfig) Empty() bool { return r == nil }

// IsDefault reports whether the GenericReconciler should be used.
func (r *ReconcileConfig) IsDefault() bool {
	if r == nil || r.Default == nil {
		return true
	}
	return *r.Default
}

// HasHooksDecl reports whether a hook declaration exists.
func (r *ReconcileConfig) HasHooksDecl() bool {
	return r != nil && r.Hooks != nil && r.Hooks.Location != ""
}

// HasConstructorDecl reports whether a constructor declaration exists.
func (r *ReconcileConfig) HasConstructorDecl() bool {
	return r != nil && r.ConstructorDecl != nil && r.ConstructorDecl.Location != ""
}

// HooksArgs returns hook args from the reconcile.hooks.args block. Nil-safe.
func (r *ReconcileConfig) HooksArgs() map[string]interface{} {
	if r == nil || r.Hooks == nil {
		return nil
	}
	return r.Hooks.Args
}

// HasRetryBackoff reports whether a retryBackoff is declared on this reconcile config's queue.
func (r *ReconcileConfig) HasRetryBackoff() bool {
	return r != nil && r.Queue.HasRetryBackoff()
}

// IsRequeueEmpty reports whether the requeue configuration is effectively empty.
func (r *ReconcileConfig) IsRequeueEmpty() bool {
	if r == nil || r.Requeue == nil {
		return true
	}
	rc := r.Requeue
	return rc.After == "" && len(rc.When) == 0 && len(rc.Or) == 0
}
