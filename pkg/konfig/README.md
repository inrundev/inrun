# pkg/konfig

`konfig` is the configuration root for the Orkestra runtime. It loads ENV variables at startup, normalises them into typed structs, and exposes them through a stable accessor API. The Katalog loader merges YAML-level values on top of what `konfig` provides, so `konfig` represents the ENV-layer defaults.

```go
kfg, err := konfig.Init()  // reads ENV, validates, returns *Konfig
```

## Precedence

```
Katalog YAML values
       ↓ merged on top of
ENV variables (konfig.Init)
       ↓ fallback to
hard-coded defaults
```

## ENV variable reference

| ENV | Accessor | Default | Description |
|-----|----------|---------|-------------|
| `ORK_ENV` | `kfg.Ork().Environment` | `development` | Runtime environment (`development`, `staging`, `production`) |
| `ORK_NAMESPACE` | `kfg.Cluster().Namespace` | `orkestra-system` | Namespace for Orkestra control-plane resources |
| `ORK_GATEWAY_ENDPOINT` | `kfg.GatewayEndpoint()` | `""` | Companion gateway URL advertised to control center |
| `ORK_SERVICE_NAME` | `kfg.Security().ServiceName` | `orkestra-runtime` | Service name used by webhook configurations |
| `KATALOG_PATH` | `kfg.Katalog().Paths()` | `[]` | Paths to Katalog YAML files |
| `QUEUE_DEPTH` | `kfg.Katalog().DefaultQueueDepth()` | `100` | Default max items per CRD reconcile queue |
| `FAILURE_THRESHOLD` | `kfg.Katalog().DefaultFailureThreshold()` | `5` | Consecutive failures before a CRD is marked degraded |
| `DEFAULT_RESYNC` | `kfg.Katalog().DefaultResync()` | `15s` | Default resync interval when not set on the CRD |
| `DEFAULT_WORKERS` | `kfg.Katalog().DefaultWorkers()` | `3` | Default worker count per CRD |
| `ORK_PORT` | `kfg.Health().Port` | `8080` | Health server port |
| `LEASE_DURATION` | `kfg.Konductor().LeaseDuration()` | `60s` | Leader election lease duration |
| `RENEW_DEADLINE` | `kfg.Konductor().RenewDeadline()` | `40s` | Leader election renew deadline |
| `RETRY_PERIOD` | `kfg.Konductor().RetryPeriod()` | `10s` | Leader election retry period |
| `ENABLE_DELETION_PROTECTION` | `kfg.Security().DeletionProtection.Enabled` | `false` | Enable deletion-protection admission webhook |
| `DELETION_PROTECTION_POLICY` | `kfg.Security().DeletionProtection.FailurePolicy` | `Fail` | Webhook failure policy |
| `ENABLE_ADMISSION_WEBHOOK` | `kfg.Security().Webhooks.Admission.Enabled` | `false` | Enable admission mutation webhook |
| `ENABLE_CONVERSION` | `kfg.Security().Conversion.Enabled` | `false` | Enable CRD version conversion webhook |
| `ENABLE_NAMESPACE_PROTECTION` | `kfg.Security().NamespaceProtection.Enabled` | `false` | Enable namespace restriction webhook |
| `TLS_CERT` / `TLS_KEY` | `kfg.Security().Webhooks.TLSCert/TLSKey` | `""` | Override TLS paths (omit to let Orkestra generate its own) |

Duration ENV vars are in whole seconds (`DEFAULT_RESYNC=15` means 15s).

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand Init(), .env loading, and namespace resolution | [docs/01-init.md](docs/01-init.md) |
| Understand SecurityConfig and its ENV mappings | [docs/02-security.md](docs/02-security.md) |
