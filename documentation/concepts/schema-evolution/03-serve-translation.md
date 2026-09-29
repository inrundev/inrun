# serve.fields.values

`serve.fields.values` runs at the Gateway before the CR is written. Callers submit a simplified intent; the Katalog fans the fields out to the CRD's internal schema. The CRD never sees the caller's vocabulary — the translation is declared once in the serve block and runs transparently on every apply.

Use it when the caller should not be coupled to the CRD schema. The CRD is an implementation detail; the intent is the contract.

---

## How it works

Declare `serve.fields` in the Katalog. Each field entry maps one intent field to one or more CRD paths via `values:`. The expressions run against `.value` — the raw value the caller submitted.

```yaml
serve:
  fields:
    schedule:
      values:
        schedule.minute:     '{{ cronMinute .value }}'
        schedule.hour:       '{{ cronHour   .value }}'
        schedule.dayOfMonth: '{{ cronDom    .value }}'
        schedule.month:      '{{ cronMonth  .value }}'
        schedule.dayOfWeek:  '{{ cronDow    .value }}'
```

The caller submits:

```yaml
target: cronjob-tutorial
name: daily-backup
schedule: "0 2 * * 1-5"
image: "gcr.io/google-containers/busybox:latest"
```

What reaches the API server:

```yaml
spec:
  schedule:
    minute:     "0"
    hour:       "2"
    dayOfMonth: "*"
    month:      "*"
    dayOfWeek:  "1-5"
  image: gcr.io/google-containers/busybox:latest
```

The structured schedule is reconstructed from the cron string entirely within the serve layer. Nothing downstream — not the CRD schema, not the reconciler, not etcd — ever sees the flat string.

---

## The pipeline order

```text
Intent submitted (ork serve apply)
    ↓
serve.fields.values   ← flat intent → CRD-shaped spec fields (at the Gateway)
    ↓
validation            ← intent gate fires on the raw request fields
    ↓
CR written to API server  ← structured spec only; caller's vocabulary gone
    ↓
normalize / mutation / reconcile  ← see the CRD shape throughout
```

`serve.fields.values` runs before the CR reaches the API server. Validation rules on `request.*` fields fire against the caller's raw input — before the fanout — so error messages speak the caller's vocabulary, not the CRD's.

```yaml
admission:
  validation:
    rules:
      - field: request.schedule
        operator: exists
        message: "schedule is required — use a cron expression (e.g. \"*/5 * * * *\")"
        action: deny
```

---

## One field to many CRD paths

A single intent field can fan out to any number of CRD paths. Each key under `values:` is a dot-notation path into `spec`:

```yaml
serve:
  fields:
    schedule:
      values:
        schedule.minute:     '{{ cronMinute .value }}'
        schedule.hour:       '{{ cronHour   .value }}'
        schedule.dayOfMonth: '{{ cronDom    .value }}'
        schedule.month:      '{{ cronMonth  .value }}'
        schedule.dayOfWeek:  '{{ cronDow    .value }}'
```

Five CRD fields from one intent field. The template functions (`cronMinute`, `cronHour`, `cronDom`, `cronMonth`, `cronDow`) each extract one component from the cron string.

---

## Comparing the three approaches

| | `normalize:` | `conversion.paths:` | `serve.fields.values` |
|---|---|---|---|
| **Translation point** | Reconciler | API server (`/convert`) | Gateway (before CR is written) |
| **CRD versions** | One | Two or more | One |
| **Caller submits via** | `kubectl apply` | `kubectl apply` | `ork serve apply` |
| **Caller sees CRD schema** | Yes | Yes | No |
| **Gateway required** | No | Yes | Yes |

`serve.fields.values` is the only approach where the CRD schema is entirely hidden from callers. The other two require the caller to know the CRD's field names, even if the format is flexible.

---

## Try it

```bash
ork init --pack use-cases/crd-conversion/with-serve-translation
cd with-serve-translation
```

Test the field fanout locally without a cluster:

```bash
ork serve play -i intent.yaml -t dev
```

Prints the built CR with `spec.schedule` as the structured object. Try an invalid cron string:

```bash
ork serve play -i intent-invalid.yaml -t dev
```

The intent gate fires on `request.schedule` and returns the error in the caller's vocabulary.

---

## Where to go next

- **[Schema Evolution](./index.md)** — pick-one table
- **[Self-service and target mode](../self-service/index.md)** — how `ork serve apply` and the Gateway delivery layer work
- **[serve.fields schema](../../reference/schema/02-katalog/22-serve-field-translation.md)** — full `serve.fields.values` reference and all field config options
