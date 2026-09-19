# Overrides

`spec.crds` in a Komposer is not for defining new operators — it is for environment-specific tuning on top of imported definitions. It uses the same field structure as a Katalog `spec.crds` entry, but any field you set here wins over the imported value.

---

## What can be overridden

Every field on a CRD entry can be overridden. You only need to declare the fields you want to change — everything else is inherited from the import source.

```yaml
spec:
  crds:
    # Override the postgres pattern for production
    postgres:
      operatorBox:
        reconcile:
          workers: 8       # default from import: 2
          resync: 30s      # default from import: 1m
    # Override the website operator
    website:
      operatorBox:
        reconcile:
          workers: 6
          resync: 15s
    # Disable an operator entirely
    database:
      enabled: false
```

The `postgres`, `website`, and `database` CRD names must have been declared in an import source. If a name in `spec.crds` does not match any import, it is a validation error.

---

## Common override fields

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | `false` suppresses the operator without removing the import. |
| `operatorBox.reconcile.workers` | int | Number of reconcile worker goroutines. |
| `operatorBox.reconcile.resync` | duration | Forced resync interval (e.g. `30s`, `5m`). |
| `finalizers` | list | Additional finalizers applied to all CRs of this kind. |

Any field valid in a Katalog CRD entry is valid here. Override only what differs between environments — leave the rest to the import source.

---

## Resolution order

```text
1. imports.registry   — pulled first
2. imports.files      — loaded next
3. imports.helm       — rendered next
4. spec.crds          — wins on every field it declares
```

This means a production Komposer can import a staging Katalog from the registry and tighten its settings via `spec.crds` without modifying the upstream definition.

---

## Conflict rules

| Situation | Result |
|-----------|--------|
| Same CRD name in two import sources | Hard error — duplicate CRD names across imports are not allowed. |
| CRD name in `spec.crds` not in any import | Validation error — `spec.crds` is for overrides only. |
| Same field in import and `spec.crds` | `spec.crds` wins. |
| Same field in two imports | Hard error — resolve by disabling one import or moving to `spec.crds`. |

---

## Real example — 08-komposer-registry

```yaml
# examples/advanced/08-komposer-registry/komposer.yaml

imports:
  registry:
    - url: ghcr.io/orkspace/patterns/postgres@v1.0.0
      oci: true

  files:
    - ./website-katalog.yaml

  helm:
    - repo: https://github.com/orkspace/charts.git
      chart: orkestra-katalog-example
      version: v0.1.0

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
```

Try it:

```bash
ork init --pack advanced
cd advanced/08-komposer-registry
ork run -f komposer.yaml
```

---

→ Previous: [01-imports.md](01-imports.md) | [Komposer index](index.md)
