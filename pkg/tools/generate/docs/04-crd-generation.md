# 04 — CRD and Katalog scaffold generation

## CRD generation

```sh
ork generate crd -f katalog.yaml
```

`CRDGenerator` derives a `CustomResourceDefinition` manifest directly from a `CRDEntry`. It does not require a running cluster — the schema is inferred from the Katalog declarations.

### Schema inference

The spec schema is built by reading:

- **Validation rules** — `spec.validation.required` fields become `required` entries; type annotations on values drive the JSON schema type.
- **Mutation defaults** — fields with declared defaults are marked with their Go zero-value types so Kubernetes knows how to handle omitted fields.
- **Template expressions** — `extractSpecPaths` scans all template strings in the `operatorBox` for `{{ .spec.* }}` references and includes those paths in the schema.

The status schema is built from `status.fields` — each declared status field becomes a property in the status schema. Printer columns (visible in `kubectl get`) are derived from the same declarations, so `kubectl get myapp` immediately shows the fields operators care about.

### CRD conversion webhook

If the CRD declares `apiTypes.webhook: true`, the generated CRD includes a conversion webhook stanza pointing to the Orkestra gateway service. This enables multi-version CRDs where the gateway handles version translation.

### Sample CR generation

`CRGenerator` produces an example CR manifest alongside the CRD. It calls `placeholderFor` to substitute typed example values for each spec field:

| Go type | Placeholder |
|---------|-------------|
| `string` | `"example-value"` |
| `int` / `int64` | `1` |
| `bool` | `true` |
| `[]string` | `["example"]` |

The sample CR sets `metadata.name` to `<crd-name>-sample` and fills in the correct `apiVersion` and `kind` from the CRD metadata.

## Katalog scaffold

```sh
ork generate katalog [--hook | --constructor | --typed] [--security] [--notification] [--providers]
```

`KatalogScaffold` generates a commented `katalog.yaml` that the operator author fills in. It is pure YAML output — all conditional logic is resolved at generation time; no template syntax appears in the output file.

### Reconcile modes

| Flag | What it generates | When to use |
|------|------------------|-------------|
| (none) | `operatorBox.default: true` with commented template blocks | Most operators — template expressions in the Katalog drive reconciliation |
| `--hook` | Typed CRD, commented `hooks` declaration | Add Go logic alongside template reconciliation |
| `--constructor` | Typed CRD, `default: false`, commented `constructor` | Full ownership of the reconcile loop |
| `--typed` | Both hooks and constructor commented; warning printed | When you know you'll use one but haven't decided which yet |

### Optional sections

`--security` and `--providers` inject additional blocks into the generated file. These are independent of the reconcile mode and may be combined freely:

```sh
ork generate katalog --hook --security
```

Each optional block is a YAML comment scaffold — the author uncomments and fills in the values relevant to their operator.

---

**Back →** [README](../README.md)
