// pkg/merger/helper_test.go
package merger

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func boolPtr(b bool) *bool { return &b }

// ── mergeKatalogSecurity ──────────────────────────────────────────────────────

func TestMergeKatalogSecurity_BaseWinsWhenOverrideEmpty(t *testing.T) {
	base := orktypes.KatalogSecurity{ServiceName: &orktypes.ServiceName{Runtime: "base-svc"}}
	override := orktypes.KatalogSecurity{}
	result := mergeKatalogSecurity(base, override)
	if result.ServiceName == nil || result.ServiceName.Runtime != "base-svc" {
		t.Errorf("expected base ServiceName to win, got %v", result.ServiceName)
	}
}

func TestMergeKatalogSecurity_OverrideWinsServiceName(t *testing.T) {
	base := orktypes.KatalogSecurity{ServiceName: &orktypes.ServiceName{Runtime: "base"}}
	override := orktypes.KatalogSecurity{ServiceName: &orktypes.ServiceName{Runtime: "override"}}
	result := mergeKatalogSecurity(base, override)
	if result.ServiceName == nil || result.ServiceName.Runtime != "override" {
		t.Errorf("expected override ServiceName, got %v", result.ServiceName)
	}
}

func TestMergeKatalogSecurity_OverrideDeletionProtection(t *testing.T) {
	base := orktypes.KatalogSecurity{}
	dp := &orktypes.DeletionProtectionConfig{Enabled: boolPtr(true)}
	override := orktypes.KatalogSecurity{DeletionProtection: dp}
	result := mergeKatalogSecurity(base, override)
	if result.DeletionProtection == nil {
		t.Error("expected override DeletionProtection to be set")
	}
}

func TestMergeKatalogSecurity_NilOverrideDeletionProtection_KeepsBase(t *testing.T) {
	dp := &orktypes.DeletionProtectionConfig{Enabled: boolPtr(true)}
	base := orktypes.KatalogSecurity{DeletionProtection: dp}
	override := orktypes.KatalogSecurity{} // nil DeletionProtection
	result := mergeKatalogSecurity(base, override)
	if result.DeletionProtection == nil {
		t.Error("base DeletionProtection must be preserved when override is nil")
	}
}

func TestMergeKatalogSecurity_ServiceNameEmptyOverride_KeepsBase(t *testing.T) {
	base := orktypes.KatalogSecurity{ServiceName: &orktypes.ServiceName{Runtime: "my-svc"}}
	override := orktypes.KatalogSecurity{ServiceName: nil}
	result := mergeKatalogSecurity(base, override)
	if result.ServiceName == nil || result.ServiceName.Runtime != "my-svc" {
		t.Errorf("empty override ServiceName must keep base, got %v", result.ServiceName)
	}
}

// ── mergeKatalogNotification ──────────────────────────────────────────────────

// ── checkDuplicate ────────────────────────────────────────────────────────────

func TestCheckDuplicate_FirstTime_NoError(t *testing.T) {
	seen := map[string]string{}
	err := checkDuplicate(seen, "website", "source-a.yaml")
	if err != nil {
		t.Errorf("first occurrence must not error: %v", err)
	}
}

func TestCheckDuplicate_SameSource_NoError(t *testing.T) {
	seen := map[string]string{"website": "source-a.yaml"}
	err := checkDuplicate(seen, "website", "source-a.yaml")
	if err != nil {
		t.Errorf("same source must not be a duplicate error: %v", err)
	}
}

func TestCheckDuplicate_DifferentSource_Error(t *testing.T) {
	seen := map[string]string{"website": "source-a.yaml"}
	err := checkDuplicate(seen, "website", "source-b.yaml")
	if err == nil {
		t.Error("different source must return duplicate error")
	}
}
