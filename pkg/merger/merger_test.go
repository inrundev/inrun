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
notification:
  teams:
    ops:
      slack:
        - "#ops-channel"
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

func TestKomposer_InheritsUpstreamNotification(t *testing.T) {
	dir := t.TempDir()
	katalogPath := writeTempKatalog(t, dir, "upstream.yaml", upstreamKatalogYAML)

	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: my-komposer\nimports:\n  files:\n    - url: " + katalogPath + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	notif := m.ToNotification()
	if notif == nil {
		t.Fatal("expected non-nil notification from upstream Katalog")
	}
	if _, ok := notif.Teams["ops"]; !ok {
		t.Error("expected ops team from upstream Katalog notification")
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

func TestKomposer_NotificationMergedFromMultipleSources(t *testing.T) {
	dir := t.TempDir()

	src1 := writeTempKatalog(t, dir, "src1.yaml", `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: src1
notification:
  teams:
    ops:
      slack:
        - "#ops"
spec:
  crds:
    alpha:
      apiTypes:
        kind: Alpha
        group: example.io
`)

	src2 := writeTempKatalog(t, dir, "src2.yaml", `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: src2
notification:
  teams:
    platform:
      slack:
        - "#platform"
spec:
  crds:
    beta:
      apiTypes:
        kind: Beta
        group: example.io
`)

	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: multi-src\nimports:\n  files:\n    - url: " + src1 + "\n    - url: " + src2 + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	notif := m.ToNotification()
	if notif == nil {
		t.Fatal("expected non-nil notification")
	}
	if _, ok := notif.Teams["ops"]; !ok {
		t.Error("expected ops team from src1")
	}
	if _, ok := notif.Teams["platform"]; !ok {
		t.Error("expected platform team from src2")
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
