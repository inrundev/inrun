# 01 — Autoscaler Overview

## What the autoscaler does

The autoscaler adjusts three runtime parameters of an operatorbox without stopping or restarting goroutines:

| Parameter | Field | Effect |
|-----------|-------|--------|
| Worker concurrency | `do.workers` | Resizes the `ResizableSemaphore` — limits how many reconciles run simultaneously |
| Queue depth limit | `do.queueDepth` | Caps enqueue — excess items are dropped with a warning |
| Resync interval | `do.resync` | Re-enqueues all cached objects at the given rate, adding pressure to drive faster convergence |

One `Autoscaler` runs per CRD that declares `autoscale:`. It starts in a goroutine launched by `DependencyKordinator` and lives for the lifetime of the operatorbox.

## The evaluation loop

```
NewAutoscaler(crdKind, spec, baseline, target, metrics)
        │
        └── Run(ctx)
               │
               ├── ticker fires every spec.interval (default: 15s)
               │
               └── evaluate()
                      │
                      ├── conditionsMet() → true
                      │       applyOverride()   — sets workers/queueDepth/resync on target
                      │       state.OverrideActive = true
                      │
                      └── conditionsMet() → false
                              if OverrideActive:
                                  start cooldown clock (spec.cooldown, default: 2m)
                                  when cooldown elapsed → restoreBaseline()
                                  state.OverrideActive = false
```

The cooldown prevents oscillation: the override stays applied until conditions have been false continuously for the cooldown duration. The clock resets whenever conditions become true again.

## Override lifecycle

```
 conditions false          conditions true
      │                          │
      │   ┌──────────────────────┘
      │   │   applyOverride()
      │   │   OverrideActive = true
      ▼   ▼
 ┌─────────────────────┐
 │  Override active    │ ◄── ticker fires every interval
 └─────────────────────┘
         │
    conditions false
         │
    cooldown clock starts
         │
    cooldown elapsed
         │
    restoreBaseline()
    OverrideActive = false
```

## Startup and restart behaviour

The autoscaler has no persistent state. A process restart always begins from the declared baseline. If conditions are still met after restart, the next evaluation tick will re-apply the override.

On clean shutdown (ctx cancelled), `restoreBaseline()` is called so that persistent targets (e.g. a semaphore whose capacity was raised) return to the declared value before the process exits.

## Thread safety

`Autoscaler.evaluate` runs on a single goroutine. `AutoscaleTarget` methods (`ResizeWorkers`, `SetQueueDepthLimit`, `SetResyncInterval`) must be safe to call from any goroutine — The `kordinator` implements these with atomic operations and mutex-guarded semaphore resize.

---

**Next →** [02 — Conditions](02-conditions.md)
