// pkg/types/types_reconcile_config.go
package types

import "github.com/orkspace/orkestra/domain"

// RemoteReconcileType is the wire protocol used for a remote reconciler call.
type RemoteReconcileType string

const (
	RemoteReconcileTypeHTTP RemoteReconcileType = "http"
	RemoteReconcileTypeGRPC RemoteReconcileType = "grpc" // reserved
)

// String returns the string representation of a RemoteReconcileType.
func (r RemoteReconcileType) String() string { return string(r) }

// ValidRemoteReconcileTypes returns all known RemoteReconcileType values.
func ValidRemoteReconcileTypes() []string {
	return []string{string(RemoteReconcileTypeHTTP), string(RemoteReconcileTypeGRPC)}
}

// IsValidRemoteReconcileType reports whether s is a known RemoteReconcileType.
func IsValidRemoteReconcileType(s string) bool {
	switch RemoteReconcileType(s) {
	case RemoteReconcileTypeHTTP, RemoteReconcileTypeGRPC:
		return true
	}
	return false
}

// RemoteReconcilerDeclaration declares a remote HTTP reconciler.
// The kordinator POSTs a PreparedRequest to Endpoint; the response is a RemoteReconcileResult.
// Orkestra owns the queue, backoff, informer, health, and metrics. The remote service owns logic.
type RemoteReconcilerDeclaration struct {
	// Type is the wire protocol. Default http; grpc reserved.
	Type RemoteReconcileType `yaml:"type,omitempty" json:"type,omitempty"`

	// Endpoint is the URL the PreparedRequest is POSTed to. Required.
	// Template expressions are supported and evaluated per-reconcile against the CR's resolver context:
	// e.g. "http://{{ .metadata.namespace }}-svc.svc.cluster.local/reconcile"
	Endpoint string `yaml:"endpoint" json:"endpoint" validate:"required"`

	// ManagedResources is used for RBAC generation — same as constructor.
	ManagedResources []domain.ManagedResource `yaml:"managedResources,omitempty" json:"managedResources,omitempty"`

	// Args is a map of values forwarded to the remote reconciler in the PreparedRequest payload.
	// Values may contain Go template expressions evaluated per-reconcile against the CR's resolver context,
	// e.g. replicas: "{{ .spec.replicas }}". Resolved booleans, integers, and JSON values are coerced
	// to their native types before serialisation.
	Args map[string]interface{} `yaml:"args,omitempty" json:"args,omitempty"`

	// Auth declares the credential for the outbound request. Uses the same shape as ExternalCallSpec.
	Auth *ExternalAuth `yaml:"auth,omitempty" json:"auth,omitempty"`

	// Timeout caps the round-trip. Defaults to 30s when omitted.
	Timeout Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// Payload controls what is serialised into the outbound request body.
	Payload *RemotePayloadConfig `yaml:"payload,omitempty" json:"payload,omitempty"`
}

// RemotePayloadConfig controls what is included in the outbound PreparedRequest body.
type RemotePayloadConfig struct {
	// Object controls which paths are stripped from the CR object before serialisation.
	Object *RemotePayloadObjectConfig `yaml:"object,omitempty" json:"object,omitempty"`

	// Children controls which previously-created child resources are injected into the payload.
	Children *RemotePayloadChildrenConfig `yaml:"children,omitempty" json:"children,omitempty"`
}

// RemotePayloadObjectConfig strips paths from the CR object before it is sent to the reconciler.
type RemotePayloadObjectConfig struct {
	// Exclude is a list of dot-notation paths to remove from the object.
	// Example: ["metadata.managedFields", "metadata.annotations"]
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// RemotePayloadChildrenConfig controls which child resources are injected into prepared.children.
type RemotePayloadChildrenConfig struct {
	// Enabled controls whether children are injected at all.
	// Default when omitted: true. Set to false to disable children injection entirely,
	// useful for reconcilers that never create Kubernetes resources.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Resources maps each allowed resource type name to its per-type config.
	// When omitted or nil: all types declared in managedResources are injected.
	// When set: only the named types are included; each entry may override the root Exclude list.
	// Example:
	//   resources:
	//     deployment:
	//       exclude: ["spec.template.metadata.annotations"]
	//     service: {}
	Resources map[string]*RemotePayloadChildrenResourceConfig `yaml:"resources,omitempty" json:"resources,omitempty"`

	// Exclude is a list of dot-notation paths to strip from every injected child object.
	// Per-resource exclude (in resources.<type>.exclude) takes precedence for that type.
	// Example: ["metadata.managedFields"]
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// RemotePayloadChildrenResourceConfig is the per-resource-type config inside payload.children.resources.
type RemotePayloadChildrenResourceConfig struct {
	// Exclude overrides the root exclude list for this resource type.
	// When nil: the root Exclude applies. When empty slice: no paths are stripped for this type.
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// HasResources reports whether an explicit resources allow-list is declared.
func (c *RemotePayloadChildrenConfig) HasResources() bool {
	return c != nil && len(c.Resources) > 0
}

// HasExclude reports whether a root exclude list is declared.
func (c *RemotePayloadChildrenConfig) HasExclude() bool {
	return c != nil && len(c.Exclude) > 0
}

// IsEnabled reports whether children injection is active.
// Defaults to true when Enabled is nil.
func (c *RemotePayloadChildrenConfig) IsEnabled() bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

// EffectiveChildren returns the active children config for callers to use.
// Returns nil only when injection is explicitly disabled (enabled: false).
// Returns an empty config (inject all, no filtering) when no config is declared.
// All callers must go through this rather than accessing Children directly.
func (p *RemotePayloadConfig) EffectiveChildren() *RemotePayloadChildrenConfig {
	if p == nil || p.Children == nil {
		return &RemotePayloadChildrenConfig{} // default: inject all
	}
	if !p.Children.IsEnabled() {
		return nil // explicitly disabled
	}
	return p.Children
}

// ReconcileConfig declares what runs and how. Default (omitted or default: true) uses the
// generic.Reconciler driven by onCreate/onReconcile/onDelete templates. Set default: false
// and declare constructor: to bring a typed Go reconciler. Workers, resync, queue, and
// requeue tune execution regardless of which reconciler runs.
type ReconcileConfig struct {
	// --- Implementation identity ---

	// Default: true → generic.Reconciler (default when omitted).
	// false → custom reconciler via Constructor.
	Default *bool `yaml:"default,omitempty" json:"default,omitempty" validate:"omitempty"`

	// Hooks declares a Go hook function. Signature: func() domain.AnyReconcileHooks.
	Hooks *HookDeclaration `yaml:"hooks,omitempty" json:"hooks,omitempty" validate:"omitempty"`

	// Constructor declares a custom reconciler constructor. Required when Default: false.
	ConstructorDecl *ConstructorDeclaration `yaml:"constructor,omitempty" json:"constructor,omitempty" validate:"omitempty"`

	// Remote declares a remote HTTP reconciler. Required when Default: false and no constructor is set.
	Remote *RemoteReconcilerDeclaration `yaml:"remote,omitempty" json:"remote,omitempty" validate:"omitempty"`

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

// IsDefault reports whether the generic.Reconciler should be used.
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

// HasRemoteDecl reports whether a remote reconciler declaration exists.
func (r *ReconcileConfig) HasRemoteDecl() bool {
	return r != nil && r.Remote != nil
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
