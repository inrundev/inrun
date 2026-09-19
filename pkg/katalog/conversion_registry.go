// pkg/katalog/conversion_registry.go
package katalog

import (
	"sync"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ConversionRegistry is the interface used by the health server's /convert handler.
// Decoupled from the Katalog struct so the health server has no import cycle.
type ConversionRegistry interface {
	GetConversionRules(kind string) *orktypes.ConversionRules
	RegisterConversionRules(rules *orktypes.ConversionRules)
}

// InMemoryConversionRegistry holds per-Kind conversion rules.
// Safe for concurrent use — the /convert endpoint reads from multiple goroutines
// and Katalog load writes once at startup.
type InMemoryConversionRegistry struct {
	mu    sync.RWMutex
	rules map[string]*orktypes.ConversionRules
}

// NewInMemoryConversionRegistry returns an initialised registry.
func NewInMemoryConversionRegistry() *InMemoryConversionRegistry {
	return &InMemoryConversionRegistry{
		rules: make(map[string]*orktypes.ConversionRules),
	}
}

// GetConversionRules returns the rules for a given Kind.
// Returns nil when no rules are registered for that Kind.
func (r *InMemoryConversionRegistry) GetConversionRules(kind string) *orktypes.ConversionRules {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rules[kind]
}

// RegisterConversionRules stores rules for the Kind declared in rules.Kind.
// Called once per CRD entry during Katalog load.
func (r *InMemoryConversionRegistry) RegisterConversionRules(rules *orktypes.ConversionRules) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rules.Kind] = rules
}

// registerConversionRulesFromSpec builds ConversionRules from a CRDEntry
// and registers them in the registry.
//
// Called from KomposeRuntimeKatalog for every CRD entry that declares
// a conversion block. Only the CRD with storageVersion declared does this —
// other versions of the same CRD don't need conversion rules registered
// because conversion is always expressed relative to the storage version.
func (reg *InMemoryConversionRegistry) registerConversionRulesFromSpec(entry orktypes.CRDEntry) {
	if entry.EffectiveConversion() == nil || entry.IsConversionParticipant() {
		return
	}

	rules := &orktypes.ConversionRules{
		Kind:           entry.APITypes.Kind,
		StorageVersion: entry.EffectiveConversion().StorageVersion,
		Paths:          entry.EffectiveConversion().Paths,
	}

	reg.RegisterConversionRules(rules)
}

func (k *Katalog) ConversionRegistry() ConversionRegistry {
	return k.conversionRegistry
}

// Test exports
func NewInMemoryRegistryForTest() *InMemoryConversionRegistry {
	return &InMemoryConversionRegistry{
		rules: make(map[string]*orktypes.ConversionRules),
	}
}
