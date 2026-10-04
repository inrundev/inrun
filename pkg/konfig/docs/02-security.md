# 02 — SecurityConfig

`SecurityConfig` consolidates all security-related configuration for the Orkestra gateway. It is populated from ENV variables at `Init()` time, then the Katalog loader merges YAML-level values on top.

## Structure

```go
type SecurityConfig struct {
    ServiceName string  // shared by all webhook configurations

    DeletionProtection struct {
        Enabled       bool
        FailurePolicy string // "Fail" or "Ignore"
        ServiceName   string
    }
    Webhooks struct {
        Admission struct{ Enabled bool }
        FailurePolicy  string
        ServiceName    string
        TLSCert        string // path or PEM; set by certmanager after generation
        TLSKey         string
        Controller struct {
            Enabled      bool
            SyncInterval time.Duration
        }
    }
    Conversion struct {
        Enabled          bool
        ConversionWindow int
    }
    NamespaceProtection struct {
        Enabled       bool
        FailurePolicy string
        ServiceName   string
    }
}
```

## What each section controls

**DeletionProtection** — a validating admission webhook that blocks DELETE requests on resources labelled `orkestra.io/deletion-protection=true`. When `Enabled: true`, the gateway registers the webhook with the configured `FailurePolicy`. `Fail` is the safe default: if the webhook cannot be reached, Kubernetes blocks the deletion.

**Webhooks.Admission** — the mutation webhook. When enabled, Orkestra injects standard labels and annotations into CRs at admission time.

**Webhooks.TLSCert / TLSKey** — initially set from `TLS_CERT` / `TLS_KEY` ENV vars. When those vars are absent, `certmanager.EnsureCertificate()` overwrites these fields after generating and storing a self-signed bundle. Downstream components (gateway, webhook server) always read from `kfg.Security()` — they do not need to know whether TLS was provided externally or generated internally.

**Webhooks.Controller** — the background goroutine that watches webhook configurations and recreates them if they are deleted. `SyncInterval` controls how often it re-checks.

**Conversion** — enables the `/convert` endpoint for multi-version CRDs. `ConversionWindow` is the rolling window size for latency statistics.

**NamespaceProtection** — a validating webhook that prevents Orkestra-managed CRs from being created in namespaces that violate the CRD's `allowedNamespaces` or `restrictedNamespaces` declarations.

## TLS lifecycle

When no `TLS_CERT` / `TLS_KEY` are provided:

1. `Init()` leaves `Webhooks.TLSCert` and `Webhooks.TLSKey` empty.
2. At startup, the gateway calls `certmanager.EnsureCertificate()`.
3. `EnsureCertificate()` generates a self-signed CA + server cert, stores them in a Kubernetes Secret, and returns the PEM bundle.
4. The Katalog loader writes the cert paths back into `kfg.Security().Webhooks.TLSCert` and `TLSKey`.
5. The webhook server reads from `kfg.Security()` and finds the paths populated.

This means the process that generates certificates is always the process that writes back into `Konfig` — there is no shared file system dependency.

