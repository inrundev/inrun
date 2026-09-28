// pkg/health/provider_stats.go
package health

import "sync"

// ProviderStats tracks per-provider reconcile and delete call totals and errors.
// Thread-safe for concurrent updates from the template reconciler.
//
// One ProviderStats instance is created per CRD at startup and shared between:
//   - generic.Reconciler, which writes to it after each provider.Reconcile / provider.Delete call
//   - BuildCRDInfoHandler, which reads it to surface error rates in the CRD detail response
//
// Detailed per-kind breakdowns are available in Prometheus via RecordProviderReconcile.
// This struct exists for fast in-memory error rate checks without querying Prometheus.
type ProviderStats struct {
	mu      sync.RWMutex
	entries map[string]*providerEntry // key: provider name
}

type providerEntry struct {
	Total  int64
	Errors int64
}

// NewProviderStats creates a new ProviderStats instance.
func NewProviderStats() *ProviderStats {
	return &ProviderStats{
		entries: make(map[string]*providerEntry),
	}
}

// RecordSuccess records a successful provider.Reconcile call for the named provider.
func (s *ProviderStats) RecordSuccess(provider string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entry(provider).Total++
}

// RecordFailure records a failed provider.Reconcile call for the named provider.
func (s *ProviderStats) RecordFailure(provider string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entry(provider)
	e.Total++
	e.Errors++
}

// RecordDeleteSuccess records a successful provider.Delete call.
func (s *ProviderStats) RecordDeleteSuccess(provider string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entry(provider).Total++
}

// RecordDeleteFailure records a failed provider.Delete call.
func (s *ProviderStats) RecordDeleteFailure(provider string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entry(provider)
	e.Total++
	e.Errors++
}

func (s *ProviderStats) entry(provider string) *providerEntry {
	if e, ok := s.entries[provider]; ok {
		return e
	}
	e := &providerEntry{}
	s.entries[provider] = e
	return e
}

// ProviderStatEntry is one provider's snapshot — provider name, totals, error rate.
type ProviderStatEntry struct {
	Provider  string
	Total     int64
	Errors    int64
	ErrorRate float64
}

// GetSnapshot returns a point-in-time snapshot for all providers that have been called.
// Only providers that have been called at least once are included.
func (s *ProviderStats) GetSnapshot() []ProviderStatEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ProviderStatEntry, 0, len(s.entries))
	for name, e := range s.entries {
		var rate float64
		if e.Total > 0 {
			rate = float64(e.Errors) / float64(e.Total)
		}
		result = append(result, ProviderStatEntry{
			Provider:  name,
			Total:     e.Total,
			Errors:    e.Errors,
			ErrorRate: rate,
		})
	}
	return result
}
