# cmd/internal

Wiring layer for Orkestra. Assembles every komponent, threads dependencies, and hands the result to `pkg/orkestra` — the supervisor that calls `Start()` and `Stop()` in order.

Three public entrypoints:

- **`KonductRuntime`** — runtime. Reconcile loop, informers, leader election. Called by `ork run`.
- **`KonductGateway`** — production gateway. TLS, admission/conversion webhooks, and the Serve layer (Gateway API). Called by `ork gate`. Cluster-only (`//go:build gateway`).
- **`KonductGatewayDev`** — local gateway. Serve layer on plain HTTP; no TLS, no webhook server. Called by `ork gate run`. Dev builds only (`//go:build !runtime && !gateway`).

## Docs

- [01 — Overview](docs/01-overview.md)
- [02 — Runtime](docs/02-runtime.md)
- [03 — Gateway](docs/03-gateway.md)
- [04 — Security](docs/04-security.md)
