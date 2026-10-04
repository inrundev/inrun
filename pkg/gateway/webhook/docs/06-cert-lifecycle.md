# 06 — Certificate Lifecycle

This document covers the full lifecycle of Orkestra's self-signed TLS certificates: generation, reuse, rollout behaviour, and recovery. It is specific to self-signed mode — when the user provides `TLS_CERT`/`TLS_KEY`, Orkestra does not manage the cert lifecycle at all.

---

## The certificate as a shared resource

The TLS Secret (`orkestra-internal-tls`) is not pod-local. It lives in the cluster and is shared across all Orkestra pods — including concurrent pods during a rolling restart. Its content determines the `caBundle` injected into every webhook configuration. If the Secret is deleted or replaced with new cert material while a pod is running, the webhook configurations become invalid until they are reconciled with the new caBundle.

This is the central constraint that shapes all lifecycle decisions below.

---

## Normal startup

```
Pod starts
  → ensureSecurity()
      → certmanager.EnsureCertificate()
          → Secret exists and is valid?
              YES → read bundle from Secret, return it
              NO  → generate new bundle, create Secret, return it
      → writeTLSToFiles(bundle)   ← cert+key to /tmp
      → return certFile, keyFile, certMgr, bundle
  → ws.SetCertManager(certMgr)
  → ws.SetCertBundle(bundle...)   ← stored in memory for housekeeper
  → ws.Start()
      → HTTPS server starts with cert from /tmp
      → housekeeper starts
          → reconcileCertSecret()  ← Secret exists? nothing to do
          → webhook configs reconciled with caBundle from cert file
```

On the first-ever startup, a new bundle is generated and stored. On every subsequent startup (including restarts), the existing Secret is reused — same cert, same caBundle, no disruption.

---

## Rolling restart (normal case)

During a rolling update, Pod B starts while Pod A is still running:

```
Pod A running (cert-A, Secret contains cert-A)
Pod B starts
  → EnsureCertificate() → Secret exists → reuses cert-A
  → Serves HTTPS with cert-A
  → Webhook configs have cert-A caBundle ✓
Pod A shuts down (cleanupOnShutdown: false)
  → Secret left intact
Pod B running (cert-A, Secret contains cert-A) ✓
```

No disruption. The cert is the same across both pods because `EnsureCertificate` is idempotent: it only generates when no valid Secret exists.

---

## Rolling restart with cleanupOnShutdown: true

`cleanupOnShutdown: true` tells the outgoing pod to delete the Secret on shutdown. This is correct for clean teardown (e.g. deleting the Orkestra installation) but creates a hazard during rolling restarts:

```
Pod A running (cert-A, Secret contains cert-A)
Pod B starts
  → EnsureCertificate() → Secret exists → reuses cert-A ✓
  → Serves HTTPS with cert-A
Pod A shuts down with cleanupOnShutdown: true
  → Deletes Secret ← cert-A is now gone from the cluster
Pod B still running (cert-A on disk, no Secret in cluster)
  → Webhook configs still have cert-A caBundle ✓ (running fine)
Pod C starts (e.g. another restart or new replica)
  → EnsureCertificate() → Secret not found → generates cert-C (NEW)
  → Serves HTTPS with cert-C
  → Webhook configs still have cert-A caBundle ✗ MISMATCH
  → All webhook calls fail: "Bad certificate"
  → Housekeeper reconciles → updates configs with cert-C caBundle
  → Recovery window: up to one safety ticker interval (default 30s)
```

The window of failure is bounded by the safety ticker interval. The housekeeper's `reconcileCertSecret` and `watchCertSecret` shrink it further — see below.

---

## Housekeeper recovery

The housekeeper addresses the `cleanupOnShutdown` hazard in two ways:

### Watch-triggered recovery

`watchCertSecret` opens a Kubernetes Watch on the TLS Secret. When a `DELETED` event arrives — meaning an outgoing pod just cleaned up — it fires the trigger channel immediately:

```
Pod A deletes Secret
  → DELETED event arrives in Pod B's watchCertSecret goroutine
  → trigger ← struct{}{}
  → reconcileAll() runs
      → reconcileCertSecret()
          → Secret not found
          → re-create from in-memory bundle (cert-A bytes stored at startup)
          → Secret restored with cert-A
  → Pod C starts → EnsureCertificate() → Secret exists → reuses cert-A ✓
```

Recovery time: one API server round-trip after the deletion event.

### Safety ticker backstop

If the Watch stream drops silently (common on managed clusters, token expiry, network partitions), the safety ticker fires every `HOUSEKEEPER_SYNC_INTERVAL` (default 30s) and calls `reconcileAll()`, which includes `reconcileCertSecret()`. This catches deletions that the Watch missed.

### Why restore from in-memory bundle, not regenerate

`reconcileCertSecret` deliberately re-creates the Secret from the `certSecretBundle` stored in memory at startup — it does not call `EnsureCertificate` again. This preserves continuity:

- The running pod's HTTPS server is already serving cert-A.
- Webhook configs already have cert-A's caBundle.
- Re-creating the Secret with cert-A's bytes means the next pod restart finds cert-A and reuses it.

Calling `EnsureCertificate` with a missing Secret would generate a new cert. The webhook configs would still reference the old caBundle until the next reconcile — creating the exact failure window we are trying to eliminate.

---

## reconcileCertSecret is called first

`reconcileAll` calls `reconcileCertSecret` before any webhook configuration reconciliation:

```go
func (ws *WebhookServer) reconcileAll() {
    ws.reconcileCertSecret()                  // 1. Secret must exist before caBundle is read
    ws.reconcileAdmissionWebhooks()           // 2. reads caBundle from cert file
    ws.reconcileDeletionProtectionWebhook()   // 3. reads caBundle from cert file
    ws.reconcileNamespaceProtectionWebhook()  // 4. reads caBundle from cert file
    ws.reconcileStrictModeWebhook()           // 5. reads caBundle from cert file
    ws.reconcileNamespaceLabels()             // 6. ensures deletion-protection labels on namespace
    ws.reconcileCRDConversionWebhooks()       // 7. re-patches caBundle on conversion CRDs
}
```

If the Secret is missing, all webhook reconciliations and the CRD conversion patch would read a stale cert file and inject a caBundle that no longer matches any living Secret. Restoring the Secret first ensures consistent state.

---

## Certificate validity and rotation

Generated certificates are valid for one year by default (`defaultCertValidFor = "1y"`). Orkestra rotates certificates pre-emptively via the housekeeper — no manual intervention required.

### Auto-rotation (default)

Every `reconcileAll` cycle, `reconcileCertSecret` checks whether the certificate stored in the Secret is within the rotation threshold. When it is:

```
reconcileCertSecret()
  → Get Secret → parse tls.crt → check NotAfter
  → time.Now().Add(threshold) >= NotAfter?
      NO  → nothing to do
      YES → GenerateTLSBundle() → new cert+key+CA
          → Update Secret with new material
          → Update ws.certSecretData in memory
          → log: "TLS cert rotated — new cert takes effect on next gateway restart"
```

The running HTTPS server continues serving the old certificate — it remains valid for the full threshold window. The new certificate takes effect on the next gateway pod restart, which picks up the updated Secret via `EnsureCertificate`.

Default threshold: **30 days before expiry**.

### Configuration

```yaml
security:
  certManager:
    autoRotate: true           # default: true — set false to opt out
    rotationThreshold: "30d"   # default: 30 days
    validFor: "2y"             # default: 1 year
```

ENV overrides (take precedence when `certManager` block is absent):

| ENV var | Default | Effect |
|---------|---------|--------|
| `TLS_AUTO_ROTATE` | `true` | Set `false` to disable rotation entirely |
| `TLS_ROTATION_THRESHOLD` | `30d` | Override the pre-rotation window (e.g. `60d`) |
| `TLS_VALID_FOR` | `1y` |  Overrie the certificate validity period (e.g. `2y`) |
  

Precedence: Katalog YAML > ENV > hard default.

### Why pre-emptive, not live

Live rotation (swapping the TLS cert while the server is running) requires the housekeeper to also update the webhook configurations' `caBundle` atomically with the cert swap, and to reload the HTTPS server's TLS config. The pre-emptive model avoids all of this:

- Old cert is still valid for `threshold` days after rotation.
- New cert takes effect naturally on the next scheduled gateway restart (deploy, rollout, scale event).
- No caBundle update is needed mid-run — the new caBundle is injected on the next startup.

### Opt-out

If you manage cert rotation externally (cert-manager, Vault PKI, manual scripts), set:

```yaml
security:
  certManager:
    autoRotate: false
```

or set `TLS_AUTO_ROTATE=false`. The housekeeper still watches for Secret deletion and restores the cert, but no expiry checks run.

---

## External certs

When `TLS_CERT` and `TLS_KEY` are provided:

- `ensureSecurity` returns nil `certMgr` and nil `bundle`.
- `SetCertManager` and `SetCertBundle` are not called.
- `certSecretData` is nil — the housekeeper skips `reconcileCertSecret` and `watchCertSecret` entirely.
- The user is responsible for the full cert lifecycle: rotation, caBundle injection, and cleanup.

---

## Summary

| Scenario | Behaviour |
|----------|-----------|
| First startup | New bundle generated, stored in Secret |
| Normal restart | Existing Secret reused, same cert |
| Rolling restart, cleanupOnShutdown: false | No disruption |
| Rolling restart, cleanupOnShutdown: true | Outgoing pod deletes Secret; housekeeper in incoming pod restores it within one Watch round-trip |
| Pod starts after Secret was deleted | Secret missing → housekeeper restores from in-memory bundle → next pod reuses same cert |
| Watch stream drops | Safety ticker (30s) catches the deletion and restores |
| Cert within 30 days of expiry (autoRotate: true) | Housekeeper rotates Secret; notifies a team if configured; new cert active on next gateway restart |
| autoRotate: false | No expiry check; rotation is a manual operational act |
| External certs | Orkestra does not touch the Secret |

→ Back: [04-tls.md](04-tls.md)
