# CronJob Pattern — Solution 2: Without Webhooks

**The same CronJob operator. No conversion webhook. No TLS. No multi-version CRD.**

Conversion webhooks exist to solve one problem: the API server stores objects in one version and clients request another. This approach eliminates the problem at the source. Orkestra detects which schedule format the user wrote and normalises it before any reconcile logic runs. The API server never sees two versions. There is nothing to convert.

```yaml
# Both of these are valid. The operator handles both identically.

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

---

## How it works

The Katalog has four stages, each unaware of which format the user chose:

**1. normalize** — runs first, before anything else. One line collapses both inputs into a canonical cron string:

```yaml
normalize:
  spec:
    schedule: "{{ cronFromAny .spec.schedule }}"
```

`cronFromAny` accepts a cron string or a structured map and always produces a five-field string. After this point, `.spec.schedule` is always `"0 2 * * 1-5"` — downstream logic never branches.

**2. mutation** — applies defaults to optional fields, using the already-normalised spec:

```yaml
admission:
  mutation:
    mutateFirst: true
    rules:
      - field: spec.concurrencyPolicy
        default: "Allow"
      - field: spec.successfulJobsHistoryLimit
        default: 3
      - field: spec.failedJobsHistoryLimit
        default: 1
      - field: spec.suspend
        default: false
```

**3. validation** — checks required fields after defaults have been applied:

```yaml
admission:
  validation:
    rules:
      - field: spec.image
        operator: exists
        message: "spec.image is required"
        action: deny
      - field: spec.schedule
        operator: exists
        message: "spec.schedule is required — use a cron string or structured object"
        action: deny
```

**4. onCreate** — creates the child `batch/v1 CronJob`. No `cronFromAny`, no `typeOf`, no branching — `.spec.schedule` is already a string:

```yaml
onCreate:
  cronJobs:
    - name: "{{ .metadata.name }}"
      schedule: "{{ .spec.schedule }}"
      image: "{{ .spec.image }}"
      suspend: "{{ .spec.suspend }}"
      successfulJobsHistoryLimit: "{{ .spec.successfulJobsHistoryLimit }}"
      failedJobsHistoryLimit: "{{ .spec.failedJobsHistoryLimit }}"
      concurrencyPolicy: "{{ .spec.concurrencyPolicy }}"
      reconcile: true
```

### Alternative: typeOf branching (no normalize block)

If you prefer not to use `normalize`, you can branch at reconcile time using `typeOf`:

```yaml
onReconcile:
  cronJobs:
    # Path A — cron string
    - name: "{{ .metadata.name }}"
      schedule: "{{ .spec.schedule }}"
      image: "{{ .spec.image }}"
      when:
        - field: "{{ typeString .spec.schedule }}"
          equals: "true"

    # Path B — structured object
    - name: "{{ .metadata.name }}"
      schedule: "{{ cronFromMap .spec.schedule }}"
      image: "{{ .spec.image }}"
      when:
        - field: "{{ typeMap .spec.schedule }}"
          equals: "true"
```

Only one path fires per reconcile. The created Kubernetes CronJob is identical regardless of which path ran. Use this when you want to preserve the raw schedule format in etcd and branch explicitly.

---

## Solution 1 vs Solution 2

| | Solution 1 (webhooks) | Solution 2 (no webhooks) |
|---|---|---|
| CRD versions | v1 + v2 (multi-version) | v1 only (single version) |
| Conversion webhook | Required — Orkestra `/convert` | Not needed |
| TLS certificates | Required | Not needed |
| Bidirectional read | Yes — v1 reads reconstructed | No — one format, one CRD |
| Migration from v1 objects | Automatic on read | Objects stay as-is |
| Best for | Public APIs, typed client access | Internal operators, rapid iteration |

**Choose Solution 2 when:**
- You control who creates the CRs (internal platform, CI/CD pipelines)
- You want to iterate quickly without schema version overhead
- You are starting fresh with no existing v1 objects to migrate

**Choose Solution 1 when:**
- External clients target a specific API version
- You have live v1 objects that must remain readable as v1
- You need the API server to enforce the schema at admission time

---

## Quick start

```bash
# 1. Validate
ork validate

# 2. Run the operator
ork run
# Orkestra applies the CRD and starts the operator

# 3. Apply CRs in either format
kubectl apply -f cr-string-schedule.yaml
kubectl apply -f cr-structured-schedule.yaml
```


```bash
# Verify
kubectl get cronjobs -n default
```

Expected:
```
NAME           SCHEDULE      PHASE    AGE
daily-backup   0 2 * * 1-5   Active   6s
print-hello    */1 * * * *   Active   4s
```

Both CRs show a canonical cron expression in `SCHEDULE`, regardless of which format was written.

---

## Notes used in this solution

| Note | What it does | Where |
|---|---|---|
| `cronFromAny .spec.schedule` | String or map → canonical cron string | `normalize` — the one place that handles both formats |
| `cronDescribe .spec.schedule` | Cron string → human-readable description | Status |
| `cronJobLastScheduleTime .children.cronjob` | Last time the child CronJob ran | Status |
| `cronJobNextScheduleTime .children.cronjob` | Next scheduled run time | Status |
| `boolTernary .spec.suspend "Suspended" "Active"` | Boolean-safe conditional | Phase status field |

---

## Eventually deprecating string format

If you later want to nudge users toward structured schedules, you can add a validation rule without touching etcd or bumping the CRD version:

```yaml
admission:
  validation:
    rules:
      - field: spec.schedule
        operator: typeOf
        value: map
        message: "spec.schedule must be a structured object — string format is deprecated. See migration guide."
        action: warn   # change to deny when ready
```

Old CRs continue to reconcile normally. When you are ready to enforce, change `action: warn` to `action: deny`. No stored object migration required at any point — the stored format never changes, only the admission policy does.

> **Note:** `action: warn` surfaces as an immediate warning in the `kubectl apply` response only when the Orkestra Gateway is deployed. Without it, Orkestra enforces the rule at reconcile time — the result is written as a `ValidationWarning` condition on the CR and visible in the Control Center **Conditions** tab:
> ```bash
> kubectl get cronjob <name> -o yaml | grep -A6 "ValidationWarning"
> # - type: ValidationWarning  status: "True"  reason: WarnRuleViolation
> #   message: "field \"spec.schedule\": spec.schedule must be a structured object..."
> ```
> For gateway-backed validation, see the admission example:

```bash
ork init --pack advanced
cd examples/advanced/07-validation-mutation
```

---

## E2E

Run the full lifecycle in one command — applies the string-format CR, asserts the CronJob is created and the status reflects the normalized schedule, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: CronJob CR created (string schedule)
    after: cr-applied
    timeout: 60s
    resources:
      - kind: CronJob
        name: daily-backup-string
        namespace: default

  - name: Structured schedule CR also accepted
    after: cr-applied
    timeout: 30s
    commands:
      - run: kubectl apply -f cr-structured-schedule.yaml
        exitCode: 0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Files

| File | Purpose |
|---|---|
| `crd-single.yaml` | Single-version CRD — no conversion webhook config |
| `katalog-no-webhook.yaml` | Complete operator |
| `cr-string-schedule.yaml` | CR using cron string format |
| `cr-structured-schedule.yaml` | CR using structured schedule object |
| `cleanup.sh` | Teardown — deletes CRs and CRD |

For the webhook-based solution with bidirectional API version support, see the [with-webhooks](../with-webhooks/README.md) directory.
