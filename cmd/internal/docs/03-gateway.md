# 03 — Gateway

The gateway is a minimal Orkestra process that handles TLS certificate management,
admission and conversion webhooks, and the Serve layer (Gateway API).
It does not run reconcilers, hold informer caches, or compete for the konductor lease.

Two build variants ship the gateway surface:

| Variant | Build tag | Entrypoint | Command |
|---------|-----------|------------|---------|
| Production | `gateway` | `KonductGateway` | `ork gate` |
| Local dev | `!runtime && !gateway` | `KonductGatewayDev` | `ork gate run` |

The production variant requires a live cluster (hard exit otherwise). The local
variant skips TLS and the webhook server — only the HTTP-based Serve layer runs.

## What the gateway is

Admission webhooks and conversion webhooks are served over HTTPS. The gateway
owns this surface: it starts the webhook server, loads or generates TLS
certificates, and serves `/validate`, `/mutate`, and `/convert` endpoints.

The health server also runs in the gateway so that Kubernetes readiness probes
work independently of the runtime.

## Why no leader election

Webhook servers are stateless. Every request is self-contained — the webhook
receives an `AdmissionReview` object, evaluates it against the Katalog rules,
and returns a decision. There is no in-memory state that needs to be consistent
across replicas.

This means any number of gateway replicas can run simultaneously. Kubernetes
distributes webhook requests across replicas through the standard `Service`
load-balancing mechanism. If a replica dies, another handles the next request
without any coordination step.

## Cluster-only (production)

`KonductGateway` exits immediately if it is not running inside a Kubernetes pod.
`utils.IsRunningInPod()` checks for the service account token that Kubernetes
injects into every pod. Outside a cluster there is no meaningful webhook endpoint
to serve (no Kubernetes API server to register with), and `ensureSecurity` would
fail without cluster credentials.

## Local development — `ork gate run`

`KonductGatewayDev` (`gateway_dev.go`, `//go:build !runtime && !gateway`) provides
the Serve layer locally without a cluster deployment:

- No TLS setup — runs on plain HTTP (health port, default `:8080`)
- No `WebhookServer` — `/validate`, `/mutate`, `/convert` are not served
- No `/katalog` webhook-stats routes — those depend on the webhook server
- Gateway API (`POST /api/v1/apply`, `GET /api/v1/resources/`) fully functional

Use `ork gate run -f katalog.yaml` to test serve routing and apply flows before
pushing a helm deployment. Admission and conversion webhook behaviour must still
be verified with `ork gate -f katalog.yaml --cr cr.yaml`.

## Komponent start order

```
1. HealthServer   — /ready, /livez, /katalog routes (HTTP :8080)
2. WebhookServer  — /validate, /mutate, /convert, /deletion-protection (HTTPS :8443)
3. Kubeclient     — already started during wiring, managed for Stop()
```

`ensureSecurity` runs before any komponent starts. It generates or loads TLS
certificates and writes the cert and key paths into `kfg` so the webhook server
picks them up when it binds its HTTPS listener. See [04-security.md](04-security.md)
for the full security wiring sequence.

## /katalog API

The gateway serves its own `/katalog` endpoint on the HTTP health server. Unlike
the runtime's `/katalog`, it contains only the stats the gateway owns:

| Endpoint | Content |
|----------|---------|
| `/katalog` | Top-level: feature flags, per-CRD webhook stats, infra protection stats |
| `/katalog/{crd}` | Per-CRD: admission, conversion, deletion protection, namespace protection |

The control center discovers the gateway URL via the `"gatewayEndpoint"` field
in the runtime's `/katalog` response (set by `ORK_GATEWAY_ENDPOINT` on the
runtime). Stats are merged per CRD by GVR key.

See [pkg/kordinator/docs/07-gateway-stats.md](../../../pkg/kordinator/docs/07-gateway-stats.md)
for the full design.

→ Back: [02-runtime.md](02-runtime.md)
→ Next: [04-security.md](04-security.md)
