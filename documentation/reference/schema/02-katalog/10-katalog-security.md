# security

Controls deletion protection, namespace protection, admission webhooks, and gateway topology for the Katalog.

```yaml
security:
  serviceName: orkestra-svc    # Kubernetes Service where Orkestra is deployed
  gatewayEndpoint: "http://orkestra-gateway.orkestra-system.svc:8080"
  namespaceProtection:
    enabled: true
    restrictedNamespaces:
      - kube-system
      - production
    allowedNamespaces:
      - dev
      - staging
    failurePolicy: Fail
    cleanupOnShutdown: false
  admission:
    webhooks:
      admission:
        enabled: true
      failurePolicy: Fail
      serviceName: orkestra-svc
      cleanupOnShutdown: false
    conversion:
      enabled: true
      conversionWindow: 100
  operatorBox:
    runtime:
      deletionProtection:
        enabled: true
        failurePolicy: Fail
        cleanupOnShutdown: false
        strictMode: false        # set true to block label removal too
```

## Top-level fields

| Field | Default | Description |
|-------|---------|-------------|
| `serviceName` | `ORK_SERVICE_NAME` env / `"orkestra"` | Kubernetes Service where Orkestra is deployed. Shared across deletion protection, namespace protection, and admission webhooks. |
| `gatewayEndpoint` | `ORK_GATEWAY_ENDPOINT` env / `""` | HTTP base URL of the companion gateway process. The runtime advertises this in its `/katalog` response so the control center can discover and merge gateway stats. Empty = no gateway configured. |

## `deletionProtection`

Registers a `ValidatingWebhookConfiguration` that blocks deletion of CRs managed by this Katalog.

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `true` (when block declared) | Activate the deletion protection webhook. |
| `failurePolicy` | `Fail` | `Fail` — block deletion on webhook error; `Ignore` — allow deletion on error. |
| `cleanupOnShutdown` | `false` | Delete the `ValidatingWebhookConfiguration` on graceful shutdown. |
| `strictMode` | `false` | When `true`, removing the `orkestra.io/deletion-protection` label from a resource is treated as a deletion attempt and blocked. To disable, set `strictMode: false` in the Katalog and restart Orkestra. |

## `namespaceProtection`

Blocks CRs from being created in forbidden namespaces via a `ValidatingWebhookConfiguration`.

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `true` (when block declared) | Activate namespace protection. |
| `restrictedNamespaces` | — | List of namespaces where CRs are **denied**. |
| `allowedNamespaces` | — | List of namespaces where CRs are **allowed**. All others are denied. |
| `failurePolicy` | `Fail` | `Fail` or `Ignore` on webhook error. |
| `cleanupOnShutdown` | `false` | Delete the webhook config on graceful shutdown. |

Declare `restrictedNamespaces` OR `allowedNamespaces` — not both. Override per-CRD via `crd-entry.md#restrictedNamespaces`.

## `webhooks`

Global admission webhook settings used by `validation` and `mutation` rules.

| Field | Default | Description |
|-------|---------|-------------|
| `admission.enabled` | `false` | Register `ValidatingWebhookConfiguration` for declarative rules. |
| `failurePolicy` | `Fail` | `Fail` or `Ignore` on webhook error. |
| `serviceName` | — | Kubernetes Service the webhook calls back to. |
| `cleanupOnShutdown` | `false` | Delete webhook config on shutdown. |

Per-CRD overrides: `spec.crds.<name>.webhooks`.

## `conversion`

Enables the `/convert` endpoint for multi-version CRD support.

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | Register the `/convert` endpoint. |
| `conversionWindow` | `100` | Rolling window size for conversion stats. |

Requires `conversion` to be declared on the CRD entry. → See [conversion.md](09-conversion.md).

---
