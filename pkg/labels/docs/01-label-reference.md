# 01 — Label Reference

Every label key, annotation key, and finalizer defined in `labels.go`, with the value it carries, who writes it, and who reads it.

---

## Deletion protection labels

### `orkestra.io/deletion-protection`

| | |
|---|---|
| **Constant** | `DeletionProtectionLabel` |
| **Value** | `"true"` (`DeletionProtectionValue`) |
| **Written by** | `Manager.EnsureDeletionProtectionLabel` (runtime reconciler), Helm chart (for Orkestra's own infrastructure), user (in gateway-only mode) |
| **Read by** | Deletion-protection admission webhook (blocks DELETE when present), strict-mode admission webhook (triggers strict-mode check when present) |

Marks a resource as protected from deletion. Any resource carrying this label — regardless of kind — is matched by the deletion-protection webhook. The reconciler adds it when `security.deletionProtection.enabled: true` and the CRD has `protectCRs: true` (or the default). The Helm chart pre-applies it to all of Orkestra's own Kubernetes resources.

Removing this label is normally possible via `kubectl label`. Under strict mode, removal is blocked unless the resource also carries the exemption label.

---

### `orkestra.io/strict-mode-exempt`

| | |
|---|---|
| **Constant** | `StrictModeExemptKey` |
| **Value** | `"true"` (`StrictModeExemptValue`) |
| **Written by** | `Manager.EnsureStrictModeExemptLabel` (runtime reconciler only) |
| **Read by** | Strict-mode admission webhook (`UPDATE` handler) |

Signals that a CRD has opted out of strict-mode enforcement via `deletionProtection.strictMode: false` in the Katalog. When the strict-mode webhook intercepts an UPDATE that removes the deletion-protection label, it checks this key in the new object. If present, the removal is allowed; if absent, it is denied.

The reconciler manages this label automatically. Manual additions are treated as drift and corrected on the next reconcile cycle. See [02 — Manager](02-manager.md) for the two-phase removal sequence.

---

## Ownership labels

### `orkestra.orkspace.io/managed`

| | |
|---|---|
| **Constant** | `ManagedKey` |
| **Value** | `"true"` (`ManagedValue`) |
| **Written by** | `Manager.EnsureManagedLabel` (runtime reconciler) |
| **Read by** | Webhook selectors, CLI filters, Control Center, resource ownership logic |

Applied to every CR that Orkestra actively manages. Used by the admission webhook selector to scope mutations and validations to Orkestra-managed resources. Used by the CLI and Control Center to filter views. The `app.kubernetes.io/tag: orkestra-internal` label on Orkestra's own infrastructure serves a separate purpose — that is used by the webhook self-healing controller to scope Watch streams.

---

### `orkestra-owner`

| | |
|---|---|
| **Constant** | `OrkestraOwner` |
| **Value** | The name of the owning CR |
| **Written by** | Resource runners (`run_deployments.go`, `run_services.go`, etc.) |
| **Read by** | Garbage collection logic, deletion path |

Applied to child resources (Deployments, Services, ConfigMaps, etc.) to identify which CR owns them. Used during CR deletion to find and clean up all child resources.

---

## Annotations

### `orkestra.orkspace.io/managed-by`

| | |
|---|---|
| **Constant** | `AnnotationManagedBy` |
| **Value** | The Katalog name (e.g. `"platform-operator"`) |
| **Written by** | `Manager.EnsureManagedAnnotations` — write-once, never overwritten |
| **Read by** | CLI (`ork describe`), Control Center, debugging |

Records which operator instance first took ownership of the resource. Useful in multi-operator environments where different controllers manage different CRDs. The value does not change even if the Katalog is renamed.

---

### `orkestra.orkspace.io/managed-since`

| | |
|---|---|
| **Constant** | `AnnotationManagedSince` |
| **Value** | RFC 3339 UTC timestamp |
| **Written by** | `Manager.EnsureManagedAnnotations` — write-once, never overwritten |
| **Read by** | CLI, Control Center, auditing |

Records when Orkestra first took ownership of the resource. This is a stable audit trail timestamp: it is written once on the first reconcile cycle and never updated, even if the CR is modified many times after.

---

## Finalizers

### `orkestra.orkspace.io/finalizer`

| | |
|---|---|
| **Constant** | `FinalizerOrkestra` |
| **Written by** | `Generic Reconciler.ensureFinalizers` |
| **Read by** | Kubernetes API server (blocks deletion until removed), `Generic Reconciler.removeFinalizers` |

Applied when a CRD declares `operatorBox.finalizers` in the Katalog. Blocks Kubernetes from garbage-collecting the CR until the reconciler has completed its cleanup logic (running `onDelete:` hooks and ordered deletion). The reconciler removes the finalizer after cleanup succeeds.

---

## Label set constructors

```go
// OrkestraBaseLabels returns a copy of the standard control-plane label set:
//   app.kubernetes.io/name: orkestra
//   app.kubernetes.io/tag:  orkestra-internal
//   orkestra.io/deletion-protection: true
//
// Used by generators and the Helm chart for Orkestra's own Kubernetes resources.
labels.OrkestraBaseLabels() map[string]string

// WithDeletionProtection returns a copy of m with the deletion-protection label
// added. The input map is never modified.
labels.WithDeletionProtection(m map[string]string) map[string]string
```

## Selector constructors

```go
// DeletionProtectionSelector returns a LabelSelector matching only:
//   orkestra.io/deletion-protection=true
// Used when building webhook ObjectSelectors.
labels.DeletionProtectionSelector() *metav1.LabelSelector

// OrkestraResourceSelector returns a LabelSelector matching all Orkestra
// control-plane resources (the app.kubernetes.io/tag=orkestra-internal set).
// Used by the webhook self-healing controller's Watch streams.
labels.OrkestraResourceSelector() *metav1.LabelSelector
```

---

**Next →** [02 — Manager](02-manager.md)
