# Katalog

!!! note "Katalog is not a Kubernetes CRD"
    `kubectl apply` will not work. Orkestra kinds are consumed by the `ork` CLI and runtime — not by the Kubernetes API server. [Your CRD is enough](/blog/your-crd-is-enough/).

A Katalog declares one or more CRDs and defines how Orkestra manages them. It is the **unit of operator definition** — everything the runtime needs to run an operator from a single YAML file.

```text
Motif     — named inputs + resource blocks. One concern.
    ↓
Katalog   — operator declaration. Imports Motifs. Defines CRDs.
    ↓
Komposer  — platform declaration. Composes Katalogs.
    ↓
E2E       — declarative end-to-end test for a Katalog.
```

---

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog

metadata:
  name: my-operator
  description: string          # optional

spec:
  finalizers:                  # optional — applied to every CRD
    - platform.example.io/cleanup

  crds:                        # required — map of CRD entries by name
    <name>:
      apiTypes:                # GVK declaration
        ...
      operatorBox:             # resource templates and reconcile logic
        ...
      imports:                 # Motif imports
        ...

profiles:                      # optional — user-defined named profiles
  networkPolicies:
    - name: allow-monitoring
      ...
  resourceQuotas:
    - name: team-medium
      ...

security:                      # optional
  ...

notification:                  # optional
  ...

providers:                     # optional
  - ...
```

---

## Try it

```bash
ork init --pack beginner
cd beginner/01-hello-website
ork run
```

This scaffolds the simplest Katalog — a single CRD that creates a Deployment and Service from a CR apply. The full reference for every field is in the pages below.

---

## Where to go

| Page | Covers |
|------|--------|
| [01-top-level.md](01-top-level.md) | Top-level Katalog structure — `metadata`, `spec.finalizers`, `spec.crds` |
| [02-crd-entry.md](02-crd-entry.md) | Fields inside `spec.crds.<name>` — enabled, workers, resync, imports |
| [03-apitypes.md](03-apitypes.md) | `apiTypes` — group, kind, version, plural, typed mode |
| [04-operatorbox.md](04-operatorbox.md) | `operatorBox` — resource templates, reconciliation strategy, pre-reconcile gates |
| [05-status.md](05-status.md) | `status` — fields written to CR status after reconcile |
| [06-when-conditions.md](06-when-conditions.md) | `when` / `or` — conditional resource creation |
| [07-validation.md](07-validation.md) | `validation` — admission rules |
| [08-mutation.md](08-mutation.md) | `mutation` — admission defaults and overrides |
| [09-conversion.md](09-conversion.md) | `conversion` — multi-version CRD support |
| [10-katalog-security.md](10-katalog-security.md) | `security` block |
| [11-katalog-notification.md](11-katalog-notification.md) | `notification` block |
| [12-katalog-providers.md](12-katalog-providers.md) | `providers` block |
| [16-resource-types.md](16-resource-types.md) | Supported Kubernetes resource types |
| [Profiles concept](../../../concepts/profiles/10-user-defined-profiles.md) | `profiles:` — user-defined named profiles |
| [15-enrich.md](15-enrich.md) | `enrich` — post-reconcile enrichment |
| [16-resource-types.md](16-resource-types.md) | Supported resource types and placeholder fields |
| [31-reconcile-remote.md](31-reconcile-remote.md) | `reconcile.remote` — remote HTTP reconciler |

---

## Where to go next

- [Motif schema](../01-motif/index.md) — reusable resource blocks imported by Katalogs
- [Komposer schema](../03-komposer/index.md) — composing multiple Katalogs
- [E2E schema](../04-e2e/index.md) — testing a Katalog
- [Orkestra Registry](../../../orkestra-registry/index.md) — publishing and consuming Katalogs
