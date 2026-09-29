# Advanced Pack

Every production operator pattern. Some examples have sub-examples showing the same concept across different deployment topologies (in-binary, cross-binary, cross-cluster).

```bash
ork init --pack advanced
cd advanced/09-hooks
ork run
```

---

## Rules and lifecycle

| Example | What it teaches |
|---|---|
| `07-validation-mutation` | Admission-time policy without a webhook server. `deny` and `warn` rules. `default` and `override` mutation. `mutateFirst`. Two enforcement boundaries: admission and reconcile, from one declaration. |
| `16-custom-resources` | The full resource lifecycle in one place. Seven sub-examples: single child, status propagation, conditional children with `when:`, drift correction, `forEach` sharding across a list, full platform composition via Motifs, and a multi-CRD pipeline. |

---

## Composition

| Example | What it teaches |
|---|---|
| `08-komposer-registry` | Production Komposer pulling from OCI registry, local Katalog, and Helm chart sources simultaneously. Version pinning, multi-source composition, environment-specific overrides. |
| `13-dependencies` | Startup ordering with `dependsOn`. Three sub-examples: in-binary (same Orkestra process), cross-binary (different processes), cross-cluster. The `healthy` vs `started` condition difference. |
| `14-cross-operator` | CRDs reading each other's live state via `cross:`. Two operators sharing data via the informer cache — zero API server calls. Three sub-examples: in-binary, cross-binary, cross-cluster. |
| `17-apitype-override` | White-label operator pattern. Import a vendor Katalog and override only `apiTypes` — your API group, your CRD identity, identical reconcile logic. |
| `18-crd-file-komposer` | Combine `apiTypes` override and `crdFile` in one Komposer. Fully self-contained: hand it off and `ork run` handles the rest. |

---

## Escape hatches

| Example | What it teaches |
|---|---|
| `09-hooks` | When the declarative layer is not enough. Typed Go hooks called at `OnReconcile` and `OnDelete`. You write the function; Orkestra calls it at the right point. The runtime still manages informers, workqueue, health, metrics, and events. |
| `10-constructor` | Full ownership of the reconcile loop. Replace the Generic Reconciler entirely. For migrating existing Go controllers or building custom state machines. Orkestra still manages informers, workqueue, and workers. |
| `11-mixed-operator-pattern` | All three patterns in one Komposer: pure declarative, hooks, and constructor operators running side-by-side in the same runtime. |

---

## Operations

| Example | What it teaches |
|---|---|
| `12-autoscale` | Dynamic worker and resync scaling based on metrics. Five sub-examples: no-autoscaler baseline, own metrics, sibling operator metrics (in-binary), sibling operator metrics (cross-cluster), external metrics source. |
| `19-endpoint-control` | Per-CRD HTTP visibility and cross-read access control. Three sub-examples: selective health endpoint disable, full endpoint disable, and the fully-dark asymmetric pattern — one CRD reads its sibling; the sibling's read back is denied by `crossAccess: false`. Both outcomes visible in kubectl printer columns. |

---

## Tooling

| Example | What it teaches |
|---|---|
| `15-any-language` | Orkestra only cares about the final Katalog YAML. Generate it from Python, Go, or Node.js — three language examples producing identical output. |

