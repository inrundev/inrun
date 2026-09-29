package profiles

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// NetworkPolicyProfile is a named NetworkPolicy preset.
type NetworkPolicyProfile string

const (
	NetworkPolicyDenyAll        NetworkPolicyProfile = "deny-all"
	NetworkPolicyDenyAllIngress NetworkPolicyProfile = "deny-all-ingress"
	NetworkPolicyDenyAllEgress  NetworkPolicyProfile = "deny-all-egress"
	NetworkPolicyAllowSameNS    NetworkPolicyProfile = "allow-same-namespace"
	NetworkPolicyAllowDNSEgress NetworkPolicyProfile = "allow-dns-egress"
)

// NetworkPolicyExpansion is a fully expanded NetworkPolicy spec.
type NetworkPolicyExpansion struct {
	Ingress     []orktypes.NetworkPolicyIngressRule
	Egress      []orktypes.NetworkPolicyEgressRule
	PolicyTypes []string
}

// ApplyNetworkPolicyProfile expands a named profile into ingress/egress rules and policy types.
// User-defined profiles in reg are checked first; falls back to built-ins.
// Returns an error for unknown profile names.
func ApplyNetworkPolicyProfile(name string, reg *orktypes.ProfileRegistry) (*NetworkPolicyExpansion, error) {
	if reg != nil {
		if def, found := reg.LookupNetworkPolicy(name); found {
			return &NetworkPolicyExpansion{
				Ingress:     def.Ingress,
				Egress:      def.Egress,
				PolicyTypes: def.PolicyTypes,
			}, nil
		}
	}
	switch NetworkPolicyProfile(strings.ToLower(name)) {
	case NetworkPolicyDenyAll:
		// Selects all pods; empty ingress and egress slices block all traffic.
		return &NetworkPolicyExpansion{
			Ingress:     []orktypes.NetworkPolicyIngressRule{},
			Egress:      []orktypes.NetworkPolicyEgressRule{},
			PolicyTypes: []string{"Ingress", "Egress"},
		}, nil

	case NetworkPolicyDenyAllIngress:
		return &NetworkPolicyExpansion{
			Ingress:     []orktypes.NetworkPolicyIngressRule{},
			PolicyTypes: []string{"Ingress"},
		}, nil

	case NetworkPolicyDenyAllEgress:
		return &NetworkPolicyExpansion{
			Egress:      []orktypes.NetworkPolicyEgressRule{},
			PolicyTypes: []string{"Egress"},
		}, nil

	case NetworkPolicyAllowSameNS:
		// Allow ingress from any pod in the same namespace.
		return &NetworkPolicyExpansion{
			Ingress: []orktypes.NetworkPolicyIngressRule{
				{
					From: []orktypes.NetworkPolicyPeer{
						{PodSelector: map[string]string{}},
					},
				},
			},
			PolicyTypes: []string{"Ingress"},
		}, nil

	case NetworkPolicyAllowDNSEgress:
		// Allow egress on UDP/TCP port 53 to any destination (DNS resolution).
		return &NetworkPolicyExpansion{
			Egress: []orktypes.NetworkPolicyEgressRule{
				{
					Ports: []orktypes.NetworkPolicyPort{
						{Protocol: "UDP", Port: "53"},
						{Protocol: "TCP", Port: "53"},
					},
				},
			},
			PolicyTypes: []string{"Egress"},
		}, nil

	default:
		return nil, fmt.Errorf("unknown networkpolicy profile: %q — allowed: deny-all, deny-all-ingress, deny-all-egress, allow-same-namespace, allow-dns-egress", name)
	}
}

// IsValidNetworkPolicyProfile reports whether name is a recognized NetworkPolicy profile.
func IsValidNetworkPolicyProfile(name string) bool {
	switch NetworkPolicyProfile(strings.ToLower(name)) {
	case NetworkPolicyDenyAll, NetworkPolicyDenyAllIngress, NetworkPolicyDenyAllEgress,
		NetworkPolicyAllowSameNS, NetworkPolicyAllowDNSEgress:
		return true
	default:
		return false
	}
}
