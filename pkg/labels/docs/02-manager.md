# 02 — Manager

The `Manager` applies the three standard label invariants to a `domain.Object` in memory, based on the current Katalog configuration. It was originally part of the reconciler but lives here so it can be used by any layer without importing reconciler internals.

---

## Responsibilities

| Method | Invariant enforced |
|--------|-------------------|
| `EnsureManagedLabel` | `managed=true` — always present on every reconciled resource |
| `EnsureDeletionProtectionLabel` | `deletion-protection=true` — present iff global protection enabled AND CRD has `protectCRs: true` |
| `EnsureStrictModeExemptLabel` | `strict-mode-exempt=true` — present iff the CRD has `strictMode: false` (overriding global) |
| `EnsureManagedAnnotations` | `managed-by` + `managed-since` — write-once audit annotations |

Each method is a pure in-memory transformation. No API calls. The caller decides when and how to persist the result.

---

## Idempotency contract

Every `Ensure*` method follows the same contract:

1. Read the current state from the object.
2. Compute the desired state from the configuration.
3. If current == desired, **return false** (no mutation).
4. If current ≠ desired, apply the mutation and **return true**.

The return value tells the caller whether a patch is needed. If all three `Ensure*` calls return false, no label patch is sent to the API server.

---

## Caller pattern (reconciler)

The `Generic Reconciler` uses the Manager in a snapshot → mutate → patch loop:

```go
// 1. Snapshot server-side labels before any in-memory mutation.
//    This is the controller-runtime MergeFrom pattern.
serverLabels := copyStringMap(obj.GetLabels())

// 2. Apply all label invariants in memory.
labelMgr.EnsureManagedLabel(obj)
labelMgr.EnsureDeletionProtectionLabel(obj, shouldHaveProtection)
labelMgr.EnsureStrictModeExemptLabel(obj, effectiveStrict)

// 3. One atomic patch: diff serverLabels → desired.
//    PatchLabels sets absent keys to null so the server removes them.
//    No-op if nothing changed.
kube.PatchLabels(ctx, obj, gvr, serverLabels, obj.GetLabels())
```

Why snapshot first? `PatchLabels` uses JSON Merge Patch. With Merge Patch, keys absent from the patch body are **left unchanged** on the server — they are not deleted. To delete a label key the patch body must explicitly set it to `null`. `PatchLabels` computes this by diffing `serverLabels` (the snapshot, representing the current server state) against `obj.GetLabels()` (the desired state after all mutations). Keys present in the snapshot but absent in the desired state are set to `null` in the patch body.

If the snapshot were skipped and the diff were computed from the object itself (which is already mutated), `current == desired` and no `null` entries would be generated — the key would survive on the server indefinitely.

---

## The two-phase deletion-protection removal

When a CRD moves from `protectCRs: true` to `protectCRs: false`, the reconciler must remove the `deletion-protection` label. Under strict mode this creates a conflict:

- The reconciler wants to remove `deletion-protection` (shouldHaveProtection = false).
- The reconciler also wants to remove `strict-mode-exempt` (effectiveStrict = true, no per-CRD override).
- The strict-mode webhook intercepts the UPDATE and checks whether `strict-mode-exempt=true` appears in the **new** object. If both labels are removed in the same patch, the new object has neither — the exemption check fails and the webhook denies the UPDATE.

The reconciler resolves this with a two-phase sequence:

### Phase 1 — first reconcile after config change

```
Server state: {deletion-protection=true, strict-mode-exempt=true, managed=true}
              (or: {deletion-protection=true, managed=true} if exempt was not present)

shouldHaveProtection = false  → remove deletion-protection
effectiveStrict = true        → normally: remove strict-mode-exempt
currentlyProtected = true     → OVERRIDE: force effectiveStrict = false
                                (keep/add strict-mode-exempt so webhook allows the UPDATE)

Patch sent: {deletion-protection: null}
            or: {deletion-protection: null, strict-mode-exempt: "true"}

Webhook sees new object: {strict-mode-exempt=true, managed=true}
Exemption check passes → UPDATE allowed ✓
```

### Phase 2 — next reconcile cycle

```
Server state: {strict-mode-exempt=true, managed=true}
              (deletion-protection is now gone)

shouldHaveProtection = false
currentlyProtected = false    → no override; effectiveStrict = true
effectiveStrict = true        → remove strict-mode-exempt

Patch sent: {strict-mode-exempt: null}

Webhook objectSelector matches on deletion-protection=true.
The object no longer has that label → webhook is NOT called → UPDATE allowed freely ✓
```

The transition from "fully protected under strict mode" to "unprotected" takes exactly two reconcile cycles. You do not manage any of this manually. Change the Katalog; the reconciler handles the rest.

---

## Drift correction

The Manager defines the desired state. On every reconcile cycle the reconciler computes what labels the object should have and patches to match. Any manual label mutation is treated as drift and corrected.

Examples:

| Manual action | Effect on next reconcile |
|--------------|--------------------------|
| `kubectl label app my-app orkestra.io/strict-mode-exempt=true` (on a strictly protected resource) | Reconciler removes the exemption label — it should not be there |
| `kubectl label app my-app orkestra.io/deletion-protection-` (under strict mode, on a protected resource) | Webhook denies the kubectl command before it reaches the server |
| `kubectl label app my-app orkestra.io/deletion-protection-` (on a resource configured with `protectCRs: false`) | Allowed. Reconciler will not re-add it (because `protectCRs: false`) |
| `kubectl label app my-app orkestra.io/managed-` | Reconciler re-adds `managed=true` on the next cycle |

In gateway-only mode (no reconciler), drift is not corrected automatically. The labels stay whatever you set them to.

---

## Constructing a Manager

```go
mgr := labels.NewManager(labels.Config{
    Standalone:                kat.IsStandaloneGateway(),
    DeletionProtectionEnabled: kat.IsDeletionProtectionEnabled(),
})
```

`Standalone: true` suppresses finalizer management — finalizers require a controller to process them, and in gateway-only mode there is no controller. Labels and annotations are still applied normally; the Manager's label methods do not change behavior based on the standalone flag.

The Manager holds no mutable state. It is safe to construct once per reconcile cycle (cheap) or to share across goroutines.
