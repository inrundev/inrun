// pkg/merger/merger_test.go
package merger

import (
	"os"
	"path/filepath"
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func writeTempKatalog(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// ── upstream Katalog fields flow through Komposer ────────────────────────────

const upstreamKatalogYAML = `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: upstream
security:
  serviceName:
    runtime: upstream-svc
spec:
  crds:
    website:
      apiTypes:
        kind: Website
        group: example.io
`

const komposerYAML = `apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: my-komposer
imports:
  files:
    - url: %s
`

func TestKomposer_InheritsUpstreamSecurity(t *testing.T) {
	dir := t.TempDir()
	katalogPath := writeTempKatalog(t, dir, "upstream.yaml", upstreamKatalogYAML)

	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: my-komposer\nimports:\n  files:\n    - url: " + katalogPath + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	sec := m.ToSecurity()
	if sec.ServiceName == nil || sec.ServiceName.Runtime != "upstream-svc" {
		t.Errorf("expected security.serviceName.runtime=upstream-svc, got %v", sec.ServiceName)
	}
}

func TestKomposer_OwnSecurityWinsOverUpstream(t *testing.T) {
	dir := t.TempDir()
	katalogPath := writeTempKatalog(t, dir, "upstream.yaml", upstreamKatalogYAML)

	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: my-komposer\nsecurity:\n  serviceName:\n    runtime: komposer-svc\nimports:\n  files:\n    - url: " + katalogPath + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	sec := m.ToSecurity()
	if sec.ServiceName == nil || sec.ServiceName.Runtime != "komposer-svc" {
		t.Errorf("expected komposer serviceName.runtime to win, got %v", sec.ServiceName)
	}
}

func TestKomposer_CRDsFromUpstreamKatalog(t *testing.T) {
	dir := t.TempDir()
	katalogPath := writeTempKatalog(t, dir, "upstream.yaml", upstreamKatalogYAML)

	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: my-komposer\nimports:\n  files:\n    - url: " + katalogPath + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	crds := m.All()
	if _, ok := crds["website"]; !ok {
		t.Error("expected website CRD from upstream Katalog")
	}
}
