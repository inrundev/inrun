# 09 — normalize: Spec Normalization

## The problem it solves

Users may express the same intent in different shapes. A CronJob operator that accepts both a plain cron string and a structured object needs to handle both — but every downstream phase (mutation, validation, template rendering, child resource creation) works best with a single canonical form.

Without normalize, you need `typeOf` branching in every `onCreate`/`onReconcile` declaration:

```yaml
# Without normalize — branching repeated everywhere the field is used
onCreate:
  cronJobs:
    - name: "{{ .metadata.name }}"
      schedule: "{{ .spec.schedule }}"
      when:
        - field: spec.schedule
          operator: typeOf
          value: string

    - name: "{{ .metadata.name }}"
      schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
      when:
        - field: spec.schedule
          operator: typeOf
          value: map
```

With normalize, the collapse happens once before anything else runs:

```yaml
# With normalize — one declaration, no branching
normalize:
  spec:
    schedule: >
      {{ if typeMap .spec.schedule }}
        {{ cronFromMap .spec.schedule }}
      {{ else }}
        {{ cronNormalize .spec.schedule }}
      {{ end }}

onCreate:
  cronJobs:
    - name: "{{ .metadata.name }}"
      schedule: "{{ .spec.schedule }}"   # always a cron string after normalize
```

## Pipeline position

Normalize runs inside `prepare.Prepare()` — step 5 of the enrichment chain — before the resolver is built, before mutation, and before validation. The reconciler receives an already-normalized object.

```text
informer cache
     │
     ▼
DeepCopy                   ← normalize never touches the cache
     │
     ▼
prepare.Prepare()
   └── applyNormalize      ← step 5: in-memory transformation, etcd unchanged
   └── base resolver       ← built from normalized object
   └── enrichment chain    ← cross, external, git all see normalized spec
   └── mutation            ← sees normalized spec
   └── validation          ← validates normalized spec
         │
         ▼
Generic Reconciler.Reconcile()
   └── onCreate / onReconcile   ← templates see canonical field values
```

## What normalize does

1. Builds a resolver from the **raw** CR (pre-normalize values visible to templates).
2. For each field declared under `normalize.spec`, evaluates the template expression.
3. Trims whitespace (YAML block scalars leave leading/trailing whitespace).
4. Parses the rendered string to the appropriate Go type (`"3"` → `int64`, `"true"` → `bool`, `"*/5 * * * *"` → `string`).
5. Writes the value back into the in-memory copy at the declared path.
6. Returns the modified copy — the informer cache object is never touched.

## Declaring normalize in a Katalog

```yaml
normalize:
  spec:
    # key: dot-notation path relative to spec
    # value: template expression — sees the raw CR
    schedule: >
      {{ if typeMap .spec.schedule }}
        {{ cronFromMap .spec.schedule }}
      {{ else }}
        {{ cronNormalize .spec.schedule }}
      {{ end }}

    # Nested paths are supported
    resources.requests.cpu: >
      {{ default .spec.resources.requests.cpu "100m" }}
```

Rules:
- Keys are paths **relative to `spec`**. The runtime prepends `spec.` automatically.
- Values are Go `text/template` expressions with the full note library available.
- Templates see the **original** field values. One-pass only — normalized field A cannot reference normalized field B.
- Empty rendered string sets the field to `""` (not nil).

## Dynamic mode only

`applyNormalize` uses `UnstructuredContent()` to write fields, which is only available on `*unstructured.Unstructured`. Typed-mode CRDs (with compiled Go types) silently skip normalize — use Go hooks for field transformation in typed mode.

All CRDs with `reconciler.default: true` (or no explicit mode) are dynamic mode and fully support normalize.

## Note functions used with normalize

| Note | Use case |
|------|----------|
| `typeMap .spec.schedule` | Check if field is a map before calling `cronFromMap` |
| `cronFromMap .spec.schedule` | Convert `{minute, hour, ...}` map to cron string |
| `cronNormalize .spec.schedule` | Expand `@daily` macros, trim whitespace, normalize 5-field string |
| `default .spec.field "fallback"` | Supply a default when field is absent |
| `typeOf .spec.field` | Return type name: `"string"`, `"map"`, `"slice"`, `"number"`, `"bool"` |

See the [note documentation](../../note/docs/README.md) for the complete function library.

## Implementation files

| File | Role |
|------|------|
| [`pkg/types/normalize.go`](../../../types/normalize.go) | `NormalizeConfig` struct — the YAML schema |
| [`pkg/runtime/kordinator/prepare/normalize.go`](../prepare/normalize.go) | `applyNormalize` — the execution engine, called at step 5 of `prepare.Prepare()` |
| [`pkg/runtime/kordinator/prepare/prepare.go`](../prepare/prepare.go) | Call site: step 5 of the enrichment chain |
