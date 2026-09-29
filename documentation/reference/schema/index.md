# Schema Reference

The schema is organised by kind. Each kind has its own subfolder.

| Kind | Folder | Description |
|------|--------|-------------|
| [Motif](01-motif/) | `01-motif/` | Reusable resource primitive. Parameterised inputs, no standalone runtime. |
| [Katalog](02-katalog/) | `02-katalog/` | Operator declaration. Defines CRDs, resources, status, and admission rules. |
| [Komposer](03-komposer/) | `03-komposer/` | Compose multiple Katalogs from files, Helm, or OCI registries. |
| [E2E](04-e2e/) | `04-e2e/` | Declarative end-to-end test for a Katalog. [spec](04-e2e/01-spec.md) · [setup](04-e2e/02-setup.md) · [expect](04-e2e/03-expect.md) |
| [Simulate](05-simulate/) | `05-simulate/` | In-memory reconciler verification — no cluster. [field reference](05-simulate/index.md) |
| [Resources](06-resources/) | `06-resources/` | Kubernetes built-ins and custom resources declarable under `onCreate`, `onReconcile`, and `onDelete`. One page per kind. |

---

## Katalog field reference

All fields that live inside a Katalog `spec.crds.<name>` entry:

| Pattern | Covers |
|----------|--------|
| [metadata](02-katalog/00-metadata/index.md) | `metadata` — name, author, version, tags, deprecation |
| [top-level](02-katalog/01-top-level.md) | Top-level Katalog structure |
| [crd-entry](02-katalog/02-crd-entry.md) | Fields inside `spec.crds.<name>` |
| [apitypes](02-katalog/03-apitypes.md) | `apiTypes` — group, kind, version, typed mode |
| [operatorbox](02-katalog/04-operatorbox.md) | `operatorBox` — reconciliation strategy |
| [status](02-katalog/05-status.md) | `status` — fields written after reconcile |
| [when-conditions](02-katalog/06-when-conditions.md) | `when` / `or` conditions |
| [validation](02-katalog/07-validation.md) | `validation` — admission rules |
| [mutation](02-katalog/08-mutation.md) | `mutation` — admission defaults and overrides |
| [conversion](02-katalog/09-conversion.md) | `conversion` — multi-version CRD support |
| [katalog-security](02-katalog/10-katalog-security.md) | `security` block |
| [katalog-notification](02-katalog/11-katalog-notification.md) | `notification` block |
| [enrich](02-katalog/15-enrich.md) | `enrich` — post-reconcile enrichment |
| [gateway-api](02-katalog/17-gateway-api.md) | `gateway.api` — Gateway API config and per-CRD `serve:` block |
