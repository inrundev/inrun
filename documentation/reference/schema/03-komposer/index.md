# Komposer

!!! note "Komposer is not a Kubernetes CRD"
    `kubectl apply` will not work. Orkestra kinds are consumed by the `ork` CLI and runtime — not by the Kubernetes API server. [Your CRD is enough](/blog/your-crd-is-enough/).

A Komposer is a Katalog with an `imports` block. It composes multiple Katalogs from different sources — OCI registry patterns, local files, or Helm charts — into a single running operator set. The `spec.crds` block provides per-environment overrides on top of the imported definitions.

```text
Motif     — named inputs + resource blocks. One concern.
    ↓
Katalog   — operator declaration. Imports Motifs.
    ↓
Komposer  — platform declaration. Composes Katalogs.
    ↓
E2E       — declarative end-to-end test for a Katalog.
```

A Komposer cannot import another Komposer — only `kind: Katalog` files are valid import targets.

---

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer

metadata:
  name: platform-full
  author: orkspace
  version: 0.1.0
  description: >
    Multi-source Komposer: OCI registry pattern + local Katalog + Helm chart.

# ── Sources ────────────────────────────────────────────────────────────────

imports:
  # OCI registry — versioned, immutable
  registry:
    - url: ghcr.io/orkspace/patterns/postgres@v1.0.0
      oci: true

  # Local Katalog — your team's custom operator
  files:
    - ./website-katalog.yaml

  # Helm chart — CRD behavior definitions bundled in a chart
  helm:
    - repo: https://github.com/orkspace/charts.git
      chart: orkestra-katalog-example
      version: v0.1.0

# ── Overrides ──────────────────────────────────────────────────────────────
# Environment-specific tuning. Wins over all sources on name conflict.

spec:
  crds:
    postgres:
      operatorBox:
        reconcile:
          workers: 8
          resync: 30s
    website:
      operatorBox:
        reconcile:
          workers: 6
          resync: 15s
    database:
      enabled: false

# Same top-level fields as Katalog
security:
  ...
notification:
  ...
```

---

## Relationship to Katalog

A Komposer shares the same top-level schema as a Katalog — `security`, `notification` — and adds one field: `imports`. The `spec.crds` block is identical in structure to a Katalog's but is used exclusively for overrides, not for defining new operators from scratch.

| | Katalog | Komposer |
|--|---------|----------|
| Defines operators | yes | via imports only |
| `spec.crds` | full CRD definitions | overrides only |
| `imports` | no | yes |
| Can be imported | yes | no |

---

## Resolution order

```text
1. imports.registry
2. imports.files
3. imports.helm
4. spec.crds   (inline overrides — wins on name conflict)
```

CRD names must be unique across all imports. A duplicate name in two different import sources is a hard error. Use `spec.crds` to override, not re-define.

---

## Top-level field accumulation

| Field | Merge strategy |
|-------|----------------|
| `security` | Komposer's setting wins on conflict. |
| `notification.teams` | Union of all imports; Komposer wins on duplicate team name. |

---

## Try it

```bash
ork init --pack advanced
cd advanced/08-komposer-registry
ork run -f komposer.yaml
```

This starts a Komposer that pulls a versioned Katalog from the Orkestra OCI registry, loads a local Katalog from a file, and renders a third from a Helm chart. The `spec.crds` block overrides `workers` and `resync` for the production environment.

---

## Where to go

| Page | Covers |
|------|--------|
| [01-imports.md](01-imports.md) | `imports.files`, `imports.helm`, `imports.registry` — all source types and auth |
| [02-overrides.md](02-overrides.md) | `spec.crds` overrides, resolution order, conflict rules |

---

## Where to go next

- [Katalog schema](../02-katalog/01-top-level.md) — the unit a Komposer imports
- [Motif schema](../01-motif/index.md) — reusable resource blocks imported by Katalogs
- [security](../02-katalog/10-katalog-security.md), [notification](../02-katalog/11-katalog-notification.md) — inherited from Katalog schema
