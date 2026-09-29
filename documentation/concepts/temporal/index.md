# Time-Dependent Workloads

Some workloads don't need a different configuration for different environments — they need a different configuration for different *times*. Dev environments should be off at night. Databases need a maintenance window. CDN edges should run more replicas when their timezone is awake.

Orkestra handles all of this declaratively. No CronJobs. No external schedulers. No custom controllers. The reconciler re-evaluates on every resync, and built-in time notes give you the vocabulary to express any time-based rule.

---

## The three primitives

### 1. `when: time:` — resource lifecycle gates

Controls whether a resource group is created or deleted based on wall-clock time (UTC):

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    reconcile: true
    when:
      - time:
          after: "09:00"
          before: "18:00"
      - dayOfWeek:
          in: [Monday, Tuesday, Wednesday, Thursday, Friday]
```

The resource is created when both conditions pass and removed automatically when either fails. On the next resync it is re-created if the window reopened. Nothing else needed.

### 2. Built-in time notes — template expressions

Five built-in notes expose time state to template expressions, status fields, and user-defined notes:

| Note | Signature | Returns |
|------|-----------|---------|
| `weekday` | `weekday` | `true` Mon–Fri UTC |
| `weekend` | `weekend` | `true` Sat–Sun UTC |
| `timeInWindow` | `timeInWindow "HH:MM" "HH:MM"` | `true` when `after ≤ now < before` UTC |
| `timeNotInWindow` | `timeNotInWindow "HH:MM" "HH:MM"` | opposite of `timeInWindow` |
| `nextCron` | `nextCron "cron-expr"` | next fire time as RFC3339 string |

`timeInWindow` and `timeNotInWindow` accept 24-hour UTC strings. They do not wrap midnight — use two separate windows if you need cross-midnight coverage.

### 3. User-defined notes — composable building blocks

Time notes become most useful when composed into domain-specific vocabulary:

```yaml
notes:
  - name: inBusinessHours
    expression: '{{ and weekday (timeInWindow "09:00" "18:00") }}'

  - name: nextBusinessHour
    expression: '{{ nextCron "0 9 * * 1-5" }}'
```

Now `inBusinessHours` and `nextBusinessHour` are available everywhere in the Katalog — in status fields, resource attributes, `when:` conditions via `field:`, and other user-defined notes.

---

## Status fields driven by time

Status fields can express the operator's current state through the same notes:

```yaml
status:
  fields:
    - path: phase
      value: "Suspended"
    - path: phase
      value: "Active"
      when:
        - field: "{{ inBusinessHours }}"
          equals: "true"
    - path: message
      value: "Suspended — resumes {{ nextBusinessHour }}"
    - path: message
      value: "Environment is running"
      when:
        - field: "{{ inBusinessHours }}"
          equals: "true"
```

The status updates on every reconcile. Control Center shows the live state without any additional tooling.

---

## Four common patterns

### Business hours — provision and deprovision on a schedule

Dev environments, staging clusters, preview apps — things that should run 9–5 Mon–Fri and nothing else. The operator creates the Deployment and Service when the window opens and removes them when it closes. No human involvement.

### Maintenance window — drain and restore on a cron

Databases need a weekly window for vacuuming, reindexing, or backups. Read replicas scale to zero, a ConfigMap signals downstream load balancers, and the replicas come back automatically when the window closes.

### Regional peak — per-timezone replica scaling

A CDN edge, API gateway, or cache layer that serves multiple timezones. Each region's deployment scales up during local business hours and scales down overnight — all from a single operator, no per-region CronJobs.

### Workload autoscaling — replica count driven by time conditions

`autoscale:` on a Deployment declaration uses `when: time:` and `when: dayOfWeek:` as scaling conditions. Instead of creating or removing the Deployment, it patches `spec.replicas` — scaling to a higher count during peak and a lower count off-peak. A `cooldown` prevents oscillation at window boundaries. See [Workload Autoscaler](../workload-autoscaler/) for the full model.

---

## How the resync drives it

All three patterns work because Orkestra's reconciler runs on a configurable resync interval:

```yaml
operatorBox:
  reconcile:
    resync: 1m
```

On every resync, the reconciler re-evaluates all notes and `when:` conditions against the current time. A resource that should no longer exist gets deleted. A resource that should exist gets created or updated. The cluster converges to the correct state for *right now*.

---

## Try it

```bash
ork init --pack use-cases/temporal
cd temporal/01-business-hours
ork simulate
ork run
```

---

## Where to go next

- [Built-in notes reference](../../reference/orkestra-notes/index.md) — full time domain notes with examples
- [Conditionals](../conditional/) — `when:` and `or:` semantics
- [User-defined notes](../notes/) — composing built-ins into domain vocabulary
- [Operator Autoscaler](../operator-autoscaler/) — time-driven worker and resync tuning
