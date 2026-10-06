package config

import "time"

// catalog.go provides accessor methods for the unexported catalogConfig
// fields. Callers should use these methods to read and mutate catalog
// configuration without exposing the underlying struct fields.
//
// Example usage:
//   kfg.Catalog().DefaultWorkers()
//   kfg.Catalog().ShutdownTimeout()
//
// The underlying catalogConfig struct is expected to use unexported field
// names (paths, defaultQueueDepth, defaultFailureThreshold, defaultResync,
// defaultWorkers, shutdownTimeout, shutdownGracePeriod).
//
// This file intentionally contains only simple, side-effect-free accessors
// and a small set of helpers for common operations.

// Paths returns the configured catalog file paths.
func (k *catalogConfig) Paths() []string {
	return k.paths
}

// HasPaths reports whether any catalog paths have been configured.
func (k *catalogConfig) HasPaths() bool {
	return len(k.paths) > 0
}

// AddPath appends a catalog path to the configuration.
func (k *catalogConfig) AddPath(path string) {
	if path == "" {
		return
	}
	k.paths = append(k.paths, path)
}

// DefaultQueueDepth returns the defaultmaximum  queue depth for CRD workers.
func (k *catalogConfig) DefaultQueueDepth() int {
	return k.defaultQueueDepth
}

// DefaultFailureThreshold returns the default degrade threshold.
func (k *catalogConfig) DefaultFailureThreshold() int {
	return k.defaultFailureThreshold
}

// DefaultResync returns the default informer resync period.
func (k *catalogConfig) DefaultResync() time.Duration {
	return k.defaultResync
}

// DefaultWorkers returns the default number of reconcile workers per CRD.
func (k *catalogConfig) DefaultWorkers() int {
	return k.defaultWorkers
}

// ShutdownTimeout returns the hard timeout used during shutdown.
func (k *catalogConfig) ShutdownTimeout() time.Duration {
	return k.shutdownTimeout
}

// ShutdownGracePeriod returns the grace period before forced shutdown.
func (k *catalogConfig) ShutdownGracePeriod() time.Duration {
	return k.shutdownGracePeriod
}

// GatewayEndpoint returns the advertised gateway endpoint; it may be empty
// when no gateway is configured.
func (k *catalogConfig) GatewayEndpoint() string {
	return k.gatewayEndpoint
}
