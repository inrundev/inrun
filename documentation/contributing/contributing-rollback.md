# Contributing: Rollback

Rollback in Orkestra is a declaration that says: when reconciliation fails too many times, re-apply the last known good spec and block forward progress until the operator fixes the declaration.

---

## The fundamental limitation today

The current rollback implementation operates at the CR level — it snapshots the CR's spec before each change and re-applies it when the trigger fires. This is correct for what it is, but it misses the most common failure mode.

CRs rarely go bad on their own. What goes bad are the **child resources** — a Deployment gets into a crash loop, a PVC becomes stuck, a Service gets a misconfigured port. These failures happen in the child resources that Orkestra created on behalf of the CR, not in the CR itself.

A rollback that only re-applies the CR spec does not help here: the CR spec was not wrong. The Deployment config was wrong. Orkestra needs to detect degradation in child resources and re-apply the previous _resource templates_, not just the previous CR spec.

This is the core design work that would make rollback genuinely useful.

---

## What is currently wired

- `pkg/types/rollback.go` — `RollbackBlock`, `RollbackTrigger` structs and YAML schema
- `pkg/runtime/reconciler/generic.go` — rollback gate (Phase 1), failure history tracking, `runRollback`, `isRollbackActive`, snapshot writes
- `pkg/runtime/kordinator/crd_health.go` — `rollbackTriggerFn` / `rollbackClearFn` callbacks so CRD health tracks rollback state

The consecutive-failure trigger and the snapshot/re-apply cycle work for reconcile-level errors. The gap is that "reconcile succeeded" and "child resources are healthy" are not the same thing.

---

## What needs design and implementation

### 1. Child resource health as a rollback signal

Today rollback triggers on reconcile errors. It should also trigger on sustained child resource degradation — a Deployment that has been unavailable for longer than a threshold, a PVC that has been pending for too long.

The `pkg/tools/plan` package tracks resource drift. The reconciler checks child status in `patchStatusWithChildren`. The missing piece is a feedback loop: if child resource health degrades persistently after a spec change, treat that as a trigger.

### 2. Template-level rollback, not spec-level

The snapshot currently captures the CR spec. A more useful snapshot would capture the resolved resource templates that were last known to produce healthy children — the actual rendered Deployment spec, Service spec, etc. Rollback would then re-apply those rendered templates rather than re-resolving the previous CR spec (which may produce the same broken output if the template logic is what broke).

### 3. Window-based trigger

`RollbackTrigger.WithinDuration` is stored but the window calculation in `shouldTriggerRollback` is incomplete. The intended behavior: trigger if N failures occur within a sliding time window. The `rollbackFailureHistory` struct has the timestamps; the calculation needs finishing.

### 4. Status surface

When rollback is active, the CR status shows only a generic error. A `RollbackActive` condition with `LastActivated` timestamp and `ConsecutiveFailures` count would make the state visible in the control center and queryable with `kubectl`.

### 5. Rollback exit event

Rollback exits when the CR generation changes. The exit is not yet recorded as a Kubernetes event or a status condition update (`RollbackActive: false`). Adding this makes the recovery observable.

### 6. Test coverage

`pkg/types/rollback_test.go` covers struct assertions. The simulate harness (`pkg/registry/simulate`) can drive full reconcile-loop tests — a test that triggers N failures and asserts rollback activation and clearing would prevent regressions.

---

## YAML shape (for reference)

```yaml
operatorBox:
  runtime:
    rollback:
      trigger:
        consecutiveFailures: 3
        withinDuration: 5m       # optional — both conditions must hold when set
      onRollback:
        deployments:
          - name: "{{ .previous.metadata.name }}"
            image: "{{ .previous.spec.image }}"
            replicas: "{{ .previous.spec.replicas }}"
            reconcile: true
```

The `.previous.*` context is hydrated from the `orkestra.orkspace.io/previous-spec` annotation, which is written before each spec change.

---

## Key files

| File | Role |
|------|------|
| `pkg/types/rollback.go` | Struct definitions and YAML schema |
| `pkg/runtime/reconciler/generic.go` | Gate (Phase 1), `runRollback`, `shouldTriggerRollback`, snapshot |
| `pkg/tools/plan/` | Resource drift and child status tracking |
| `pkg/runtime/kordinator/crd_health.go` | Health callbacks for rollback state |
