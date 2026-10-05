//go:build !runtime && !gateway

package serve

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// chainTestKatalog builds a single-CRD katalog for kind "ServiceRequest" —
// its ServeTarget() defaults to the lowercased kind, "servicerequest",
// which every test below uses directly.
func chainTestKatalog(serveName string, tokens map[string]orktypes.ServeTokenPermissions) *katalog.Katalog {
	return katalog.NewFromEntryPointers(map[string]*orktypes.CRDEntry{
		"servicerequest": {
			APITypes: orktypes.APITypes{
				Group:   "demo.orkestra.io",
				Version: "v1",
				Kind:    "ServiceRequest",
				Plural:  "servicerequests",
			},
			Serve: &orktypes.ServeConfig{
				Enabled:   true,
				Name:      serveName,
				Namespace: "default",
				Tokens:    tokens,
			},
		},
	})
}

func TestRunCreateUpdateChain_UnknownTarget(t *testing.T) {
	k := chainTestKatalog("", nil)
	_, _, _, err := runCreateUpdateChain(
		k, map[string]interface{}{"target": "does-not-exist"}, "dev", "", orktypes.ServeOpCreate,
	)
	if err == nil {
		t.Fatal("expected an error for an unknown target")
	}
}

func TestRunCreateUpdateChain_MissingTarget(t *testing.T) {
	k := chainTestKatalog("", nil)
	_, _, _, err := runCreateUpdateChain(
		k, map[string]interface{}{"name": "x"}, "dev", "", orktypes.ServeOpCreate,
	)
	if err == nil {
		t.Fatal(`expected an error when "target" is absent`)
	}
}

func TestRunCreateUpdateChain_TokenDenied(t *testing.T) {
	k := chainTestKatalog("", map[string]orktypes.ServeTokenPermissions{
		"dev": {Permissions: orktypes.ServePermissionSet{Global: []string{"get"}}}, // no create
	})
	_, _, _, err := runCreateUpdateChain(
		k, map[string]interface{}{"target": "servicerequest", "name": "x"}, "dev", "", orktypes.ServeOpCreate,
	)
	if err == nil {
		t.Fatal("expected the token to be denied create")
	}
}

func TestRunCreateUpdateChain_UnknownTokenDenied(t *testing.T) {
	// Once ANY token is declared, an unlisted token name is denied outright —
	// no implicit fallback to "allow all".
	k := chainTestKatalog("", map[string]orktypes.ServeTokenPermissions{
		"dev": {Permissions: orktypes.ServePermissionSet{Global: []string{"*"}}},
	})
	_, _, _, err := runCreateUpdateChain(
		k, map[string]interface{}{"target": "servicerequest", "name": "x"}, "someone-else", "", orktypes.ServeOpCreate,
	)
	if err == nil {
		t.Fatal("expected an unlisted token name to be denied")
	}
}

func TestRunCreateUpdateChain_BuildsCRWithProvenance(t *testing.T) {
	k := chainTestKatalog("", nil) // no token restrictions -> allow all
	obj, crd, alias, err := runCreateUpdateChain(
		k,
		map[string]interface{}{"target": "servicerequest", "name": "payments-api"},
		"payments-repo", "payments-repo", orktypes.ServeOpCreate,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.GetName() != "payments-api" || obj.GetNamespace() != "default" {
		t.Errorf("name/namespace = %s/%s", obj.GetName(), obj.GetNamespace())
	}
	if crd.Kind() != "ServiceRequest" {
		t.Errorf("Kind = %q", crd.Kind())
	}
	if alias != "" {
		t.Errorf("alias = %q, want empty (primary target)", alias)
	}
	if got := obj.GetAnnotations()["orkestra.orkspace.io/serve-source"]; got != "payments-repo" {
		t.Errorf("serve-source annotation = %q, want payments-repo", got)
	}
	if got := obj.GetAnnotations()["orkestra.orkspace.io/serve-target"]; got != "servicerequest" {
		t.Errorf("serve-target annotation = %q, want servicerequest", got)
	}
}

func TestRunCreateUpdateChain_EmptySourceOmitsServeSourceAnnotation(t *testing.T) {
	// "ork serve play" passes source="" — no caller identity beyond the token.
	k := chainTestKatalog("", nil)
	obj, _, _, err := runCreateUpdateChain(
		k, map[string]interface{}{"target": "servicerequest", "name": "x"}, "dev", "", orktypes.ServeOpCreate,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := obj.GetAnnotations()["orkestra.orkspace.io/serve-source"]; ok {
		t.Error("serve-source annotation should be absent when source is empty")
	}
}

func TestRunCreateUpdateChain_MissingNameRejected(t *testing.T) {
	// serve.name isn't declared and no raw "name" is supplied.
	k := chainTestKatalog("", nil)
	_, _, _, err := runCreateUpdateChain(
		k, map[string]interface{}{"target": "servicerequest"}, "dev", "", orktypes.ServeOpCreate,
	)
	if err == nil {
		t.Fatal("expected an error when neither serve.name nor a raw name is available")
	}
}
