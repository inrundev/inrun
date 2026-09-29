# Hooks and constructors

Typed operators — those using Go hooks or a custom constructor — can be simulated when you run simulate from your own compiled binary.

---

## How it works

When you run `ork generate registry && go build`, the `init()` function in the generated `pkg/typeregistry/zz_generated_typeregistry.go` populates two global registries at binary startup:

```go
// pkg/types/types.go
var HookRegistry      = map[schema.GroupVersionKind]func() domain.AnyReconcileHooks{}
var ReconcilerRegistry = map[schema.GroupVersionKind]NewReconcilerFunc{}
```

`ork simulate` reads these registries at runtime. If an entry exists for the CRD being simulated, it wires the real function into the reconcile loop.

---

## Typed hooks

For hook operators (`operatorBox.reconciler.hooks:`), `HookRegistry` contains the hook function. Simulate wires it into `Generic Reconciler` and calls it on every cycle against the fake cluster.

```bash
# From your operator directory, using your compiled binary
09-hooks$ ork simulate --cr cr.yaml

Simulating database/my-db

  Cycle 1:
    + deployments/my-db-postgres
    + services/my-db-svc
  ✓ Steady state at cycle 2 in 194ms
```

The hook runs its full body — `OrkestraRegistry.Create(...)` calls go to the fake cluster and are recorded as ops.

---

## Custom constructors

For constructor operators (`operatorBox.reconciler.default: false`), `ReconcilerRegistry` contains the factory function. Simulate calls it with the fake kubeclient and informer, then uses the returned reconciler for all cycles.

```bash
10-constructor$ ork simulate --cr cr.yaml

Simulating pipeline/my-pipeline

  Cycle 1:
    + jobs/my-pipeline-build
  Cycle 2:
    + jobs/my-pipeline-test
  Cycle 3:
    ~ status/my-pipeline
  ✓ Steady state at cycle 4 in 220ms
```

The constructor owns its full reconcile loop — simulate just provides the fake cluster for it to act against.

**Note:** `ev event.Recorder` passed to the constructor during simulation is `event.Discard()` — a silent no-op. All `Eventf` calls are discarded without panicking. No nil guards are needed.

---

## Standard ork binary

When running `ork simulate` from the standard `ork` binary (not a custom operator binary), both registries are empty for your custom types. Simulate falls back to `Generic Reconciler` with a nil hook binder:

- Declarative operatorBox (status fields, templates) — simulated normally
- Hook body — not executed
- Constructor body — not executed (Generic Reconciler runs the status layer instead)

This is still useful for verifying template logic before building the binary.

