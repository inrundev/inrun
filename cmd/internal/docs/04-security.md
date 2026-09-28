# 04 — Security

`internal/security.go` owns TLS provisioning and infrastructure security setup for the gateway. Two functions are called in sequence from `KonductGateway`: `ensureSecurity` (synchronous, one-shot) and `WireWebhookHousekeeperInfra` (registers callbacks, enables continuous reconciliation).

---

## What ensureSecurity does

`ensureSecurity` performs three operations in order before any komponent starts.

**1. Namespace labeling** — if deletion protection is enabled, patches the Orkestra namespace with the standard Orkestra resource labels. The deletion-protection webhook uses an `ObjectSelector` to narrow which resources it intercepts. Kubernetes namespaces carry no labels by default, so the webhook would never fire for the Orkestra namespace unless the label is applied explicitly.

**2. TLS check** — if `kat.NeedsCertificates()` is false (no deletion protection, admission webhooks, or conversion webhooks declared), returns immediately with empty paths and nil cert manager.

**3. Certificate provisioning** — if `TLS_CERT` and `TLS_KEY` are set, those paths are used as-is. Otherwise, `certmanager.EnsureCertificate` generates a self-signed bundle, stores it in the `orkestra-tls` Secret, and writes cert and key to temporary files.

**4. CRD conversion patch** — patches `spec.conversion.webhook.clientConfig.caBundle` on every CRD that declares `conversion.updateCRD: true`, using `applyCRDConversionPatch`. Required before the HTTPS server starts — the Kubernetes API server validates the webhook TLS cert during every conversion request.

```
ensureSecurity
    ├── ensureNamespaceLabeled    (deletion protection only)
    ├── kat.NeedsCertificates()   → return early if false
    ├── kfg: TLS_CERT + TLS_KEY  → return early if provided
    ├── certmanager.EnsureCertificate → orkestra-tls Secret
    ├── writeTLSToFiles           → /tmp/orkestra-tls-cert-*.pem
    └── patchConversionCRDs       → applyCRDConversionPatch per CRD
```

---

## What WireWebhookHousekeeperInfra does

`ensureSecurity` applies security state once at startup. A gap existed: a human could strip the namespace labels or the CRD caBundle after startup, and nothing would restore them until the next restart.

`WireWebhookHousekeeperInfra` closes that gap by registering two callbacks on the `WebhookServer` before `Start()` is called:

**`ConversionCRDPatchFn`** — a closure that calls `applyCRDConversionPatch` with the captured kubeclient, service name, and namespace. The housekeeper's `reconcileCRDConversionWebhooks` calls this on every reconcile cycle. A dedicated `watchSingleConversionCRD` goroutine per conversion CRD triggers an immediate reconcile on any MODIFIED event — a stripped caBundle is restored within one API round-trip.

**`CRDWatcher`** — a `crdWatcher` implementation backed by the apiextensions client. Passed to `SetCRDWatcher` so the housekeeper can open Kubernetes Watch streams on individual CRD objects without the webhook package importing `k8s.io/apiextensions-apiserver`.

The namespace label reconciler (`reconcileNamespaceLabels`) uses the existing `kubeClient kubernetes.Interface` already on `WebhookServer` — no additional callback is needed for it.

```
WireWebhookHousekeeperInfra(ws, kube, kat, kfg)
    ├── ws.SetConversionCRDPatcher(fn)
    │       fn closes over: kube.ApiextensionsClient(), serviceName, namespace
    │       fn delegates to: applyCRDConversionPatch()
    └── ws.SetCRDWatcher(&crdWatcher{kube})
            Watch() uses: kube.ApiextensionsClient().ApiextensionsV1().CRDs().Watch()
```

`applyCRDConversionPatch` is the single shared implementation used by both `patchConversionCRDs` (startup) and the housekeeper patcher (lifecycle).

---

## Why gateway-only

`ensureSecurity` and `WireWebhookHousekeeperInfra` are called only from `KonductGateway`. The runtime does not serve webhooks, has no HTTPS listener, and does not need TLS or CRD conversion patches.

---

## Skipped outside a pod

`KonductGateway` calls `utils.IsRunningInPod()` before calling `ensureSecurity`. Outside a Kubernetes pod there is no service account token, no API server to patch, and no webhook endpoint reachable by the Kubernetes control plane.

---

## Infrastructure security — what is now watched vs applied-once

| Resource | Applied at startup | Reconciled by housekeeper |
|----------|--------------------|--------------------------|
| Namespace labels | `ensureNamespaceLabeled` | `reconcileNamespaceLabels` (safety ticker) |
| CRD conversion caBundle | `patchConversionCRDs` | `reconcileCRDConversionWebhooks` (safety ticker + CRD watcher) |
| TLS Secret | `certmanager.EnsureCertificate` | `reconcileCertSecret` (watch + safety ticker) |
| Webhook configurations | `register*Webhook` in `Start()` | `reconcile*Webhook` (webhook watcher + safety ticker) |

→ Back: [01-overview.md](01-overview.md)
