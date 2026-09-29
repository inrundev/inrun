package types_test

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func crdWithNetworkPolicyOnCreate(nps ...orktypes.NetworkPolicyTemplateSource) orktypes.CRDEntry {
	return orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnCreate: &orktypes.HookTemplates{
					NetworkPolicies: nps,
				},
			},
		},
	}
}

func TestCollectNetworkPolicyProfileEntries_Empty(t *testing.T) {
	c := orktypes.CRDEntry{}
	assert.Empty(t, c.CollectNetworkPolicyProfileEntries())
}

func TestCollectNetworkPolicyProfileEntries_NoProfile(t *testing.T) {
	c := crdWithNetworkPolicyOnCreate(orktypes.NetworkPolicyTemplateSource{Name: "allow"})
	assert.Empty(t, c.CollectNetworkPolicyProfileEntries())
}

func TestCollectNetworkPolicyProfileEntries_ProfileReturned(t *testing.T) {
	c := crdWithNetworkPolicyOnCreate(orktypes.NetworkPolicyTemplateSource{
		Name:    "default",
		Profile: "deny-all",
	})
	entries := c.CollectNetworkPolicyProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "onCreate", entries[0].Phase)
	assert.Equal(t, "default", entries[0].ResourceName)
	assert.Equal(t, "deny-all", entries[0].Profile)
	assert.False(t, entries[0].Mixed)
}

func TestCollectNetworkPolicyProfileEntries_Mixed_Ingress(t *testing.T) {
	c := crdWithNetworkPolicyOnCreate(orktypes.NetworkPolicyTemplateSource{
		Name:    "np",
		Profile: "deny-all",
		Ingress: []orktypes.NetworkPolicyIngressRule{{}},
	})
	entries := c.CollectNetworkPolicyProfileEntries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Mixed)
}

func TestCollectNetworkPolicyProfileEntries_Mixed_Egress(t *testing.T) {
	c := crdWithNetworkPolicyOnCreate(orktypes.NetworkPolicyTemplateSource{
		Name:    "np",
		Profile: "allow-dns-egress",
		Egress:  []orktypes.NetworkPolicyEgressRule{{}},
	})
	entries := c.CollectNetworkPolicyProfileEntries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Mixed)
}

func TestCollectNetworkPolicyProfileEntries_Mixed_PolicyTypes(t *testing.T) {
	c := crdWithNetworkPolicyOnCreate(orktypes.NetworkPolicyTemplateSource{
		Name:        "np",
		Profile:     "deny-all",
		PolicyTypes: []string{"Ingress"},
	})
	entries := c.CollectNetworkPolicyProfileEntries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Mixed)
}

func TestCollectNetworkPolicyProfileEntries_TemplateExpr(t *testing.T) {
	c := crdWithNetworkPolicyOnCreate(orktypes.NetworkPolicyTemplateSource{
		Profile: "{{ .Spec.Profile }}",
	})
	entries := c.CollectNetworkPolicyProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "{{ .Spec.Profile }}", entries[0].Profile)
}

func TestCollectNetworkPolicyProfileEntries_OnReconcile(t *testing.T) {
	c := orktypes.CRDEntry{
		OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				OnReconcile: &orktypes.HookTemplates{
					NetworkPolicies: []orktypes.NetworkPolicyTemplateSource{
						{Name: "np", Profile: "allow-same-namespace"},
					},
				},
			},
		},
	}
	entries := c.CollectNetworkPolicyProfileEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "onReconcile", entries[0].Phase)
}
