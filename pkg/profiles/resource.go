// Package profiles provides named presets — resource, security, probe, and
// autoscaler — that expand into fully-formed Orkestra types at katalog load
// time. The runtime never sees profile names; it only ever sees the expanded
// struct as if the user had written every field manually.
//
// Both pkg/katalog (validation + expansion) and pkg/resources (runtime
// resolution) import this package. It imports only pkg/types and pkg/utils,
// keeping the dependency graph clean.
package profiles

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ResourceProfile is a named CPU/memory preset.
type ResourceProfile string

const (
	ResourceTiny         ResourceProfile = "tiny"
	ResourceSmall        ResourceProfile = "small"
	ResourceMedium       ResourceProfile = "medium"
	ResourceLarge        ResourceProfile = "large"
	ResourceBurst        ResourceProfile = "burst"
	ResourceSteady       ResourceProfile = "steady"
	ResourceComputeHeavy ResourceProfile = "compute-heavy"
	ResourceMemoryHeavy  ResourceProfile = "memory-heavy"
)

// ApplyResourceProfile expands a named resource profile into a complete
// ResourceRequirements. User-defined profiles in reg are checked first; falls
// back to built-ins. Returns an error for unknown profile names.
func ApplyResourceProfile(name string, reg *orktypes.ProfileRegistry) (*orktypes.ResourceRequirements, error) {
	if reg != nil {
		if def, found := reg.LookupResource(name); found {
			return &orktypes.ResourceRequirements{
				Requests: def.Requests,
				Limits:   def.Limits,
			}, nil
		}
	}
	switch ResourceProfile(strings.ToLower(name)) {
	case ResourceTiny:
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "25m", "memory": "64Mi"},
			Limits:   map[string]string{"cpu": "100m", "memory": "128Mi"},
		}, nil
	case ResourceSmall:
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "100m", "memory": "128Mi"},
			Limits:   map[string]string{"cpu": "500m", "memory": "512Mi"},
		}, nil
	case ResourceMedium:
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "250m", "memory": "256Mi"},
			Limits:   map[string]string{"cpu": "1", "memory": "1Gi"},
		}, nil
	case ResourceLarge:
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "500m", "memory": "512Mi"},
			Limits:   map[string]string{"cpu": "2", "memory": "2Gi"},
		}, nil
	case ResourceBurst:
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "200m", "memory": "256Mi"},
			Limits:   map[string]string{"cpu": "2", "memory": "2Gi"},
		}, nil
	case ResourceSteady:
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "300m", "memory": "256Mi"},
			Limits:   map[string]string{"cpu": "600m", "memory": "512Mi"},
		}, nil
	case ResourceComputeHeavy:
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "1", "memory": "512Mi"},
			Limits:   map[string]string{"cpu": "2", "memory": "1Gi"},
		}, nil
	case ResourceMemoryHeavy:
		return &orktypes.ResourceRequirements{
			Requests: map[string]string{"cpu": "250m", "memory": "1Gi"},
			Limits:   map[string]string{"cpu": "500m", "memory": "2Gi"},
		}, nil
	default:
		return nil, fmt.Errorf("unknown resource profile: %q — built-ins: tiny, small, medium, large, burst, steady, compute-heavy, memory-heavy", name)
	}
}

// IsValidResourceProfile reports whether name is a recognized resource profile.
func IsValidResourceProfile(name string) bool {
	switch ResourceProfile(strings.ToLower(name)) {
	case ResourceTiny, ResourceSmall, ResourceMedium, ResourceLarge,
		ResourceBurst, ResourceSteady, ResourceComputeHeavy, ResourceMemoryHeavy:
		return true
	default:
		return false
	}
}
