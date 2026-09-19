# normalize:

`normalize:` runs before mutation, validation, and reconcile. It collapses input format variation into one canonical shape — without touching etcd, without a webhook, without a multi-version CRD.

Use it when you want to accept multiple CR shapes for the same field without branching logic downstream.

---

## How it works

Declare a `normalize:` block in the Katalog alongside `mutation:` and `validation:`. Each field expression runs against the raw CR as submitted. The normalised values replace the raw values in the in-memory resolver — the stored CR is unchanged.

```yaml
normalize:
  spec:
    schedule: "{{ cronFromAny .spec.schedule }}"
```

After `normalize:` runs, `.spec.schedule` is always a five-field cron string — whether the user wrote `"0 2 * * 1-5"` or `{minute: "0", hour: "2", ...}`. Every downstream expression, validation rule, and template sees the canonical form. No branching required.

---

## The pipeline order

```text
CR submitted
    ↓
normalize:      ← raw input → canonical shape (in-memory only)
    ↓
mutation:       ← apply defaults to the normalised CR
    ↓
validation:     ← check required fields after defaults are set
    ↓
onCreate / onReconcile   ← templates see canonical fields throughout
```

`normalize:` runs before `mutation:`, so defaults apply to the normalised value. If a user submits neither a string nor a map, `normalize:` can supply a default via `{{ default "0 * * * *" (cronFromAny .spec.schedule) }}`.

---

## Accepting two schedule formats

```yaml
# Both of these work. The reconciler sees the same thing either way.

# Cron string
spec:
  schedule: "0 2 * * 1-5"

# Structured object
spec:
  schedule:
    minute: "0"
    hour: "2"
    dayOfMonth: "*"
    month: "*"
    dayOfWeek: "1-5"
```

The Katalog:

```yaml
normalize:
  spec:
    schedule: "{{ cronFromAny .spec.schedule }}"

admission:
  mutation:
    rules:
      - field: spec.concurrencyPolicy
        default: "Allow"
onCreate:
  cronJobs:
    - name: "{{ .metadata.name }}"
      schedule: "{{ .spec.schedule }}"   # always a cron string here
      image: "{{ .spec.image }}"
      reconcile: true
```

`cronFromAny` accepts a cron string or a structured map and always returns a five-field cron string. `@`-macros (`@hourly`, `@daily`) are expanded transparently.

---

## Alternative: branch at reconcile time

If you want to preserve the raw format in etcd and handle both shapes explicitly, use `typeString` and `typeMap` in `when:` conditions instead of `normalize:`:

```yaml
onReconcile:
  cronJobs:
    # Path A — cron string
    - name: "{{ .metadata.name }}"
      schedule: "{{ .spec.schedule }}"
      when:
        - field: "{{ typeString .spec.schedule }}"
          equals: "true"

    # Path B — structured object
    - name: "{{ .metadata.name }}"
      schedule: "{{ cronFromMap .spec.schedule }}"
      when:
        - field: "{{ typeMap .spec.schedule }}"
          equals: "true"
```

Only one path fires per reconcile. The child resource is identical regardless of which path ran. Use this when you want the stored format to remain as the user submitted it — for audit trails, or when the raw shape carries meaning.

---

## Deprecating a format without schema migration

When you are ready to steer users away from the string format, add a validation rule — no etcd migration, no CRD version bump:

```yaml
admission:
  validation:
    rules:
      - field: spec.schedule
        operator: typeOf
        value: map
        message: "spec.schedule as a string is deprecated — use the structured object. See migration guide."
        action: warn   # change to deny when ready
```

Old CRs continue to reconcile normally. `normalize:` still handles them. The warning surfaces at `kubectl apply` time — at the next admission call for each CR, not at migration time.

---

## Try it

```bash
ork init --pack use-cases/crd-conversion/without-webhooks
cd without-webhooks
# Follow the steps in README
```

This runs the CronJob operator accepting both schedule formats. Apply both CRs and observe that the operator reconciles identically regardless of which shape was submitted.

---

## Where to go next

- **[conversion.paths:](./02-conversion-paths.md)** — when the API server needs to know about two versions
- **[Conditionals](../conditional/index.md)** — `when:` and `or:` in depth
