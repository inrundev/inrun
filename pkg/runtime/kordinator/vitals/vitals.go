package vitals

import (
	"github.com/orkspace/orkestra/pkg/konfig"
)

// NewCRDHealth initializes a CRDHealth tracker for a given CRD name.
// The reconciler starts in an "unhealthy" state until the first successful reconcile.
func NewCRDHealth(name string) *CRDHealth {
	h := &CRDHealth{name: name}
	h.healthy.Store(false)
	h.pending.Store(true)
	h.degraded.Store(false)

	// Add katalog tracker
	h.orkHealth = NewRuntimeHealth()
	return h
}

// NewRuntimeHealth initializes a CRDHealth tracker for Orkestra
func NewRuntimeHealth() *RuntimeHealth {
	h := &RuntimeHealth{name: konfig.Ork}
	h.orkReady.Store(true)
	h.katReady.Store(false)
	return h
}
