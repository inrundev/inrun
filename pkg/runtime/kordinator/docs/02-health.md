# 02 — CRDHealth

`CRDHealth` tracks the runtime health of a single CRD. Every hot-path read and write uses `atomic` operations — no locks on the reconcile path.

## Health states

| State | Meaning |
|---|---|
| `pending` | CRD is registered but workers have not started |
| `started` | Worker goroutines are running |
| `healthy` | At least one reconcile completed without error |
| `degraded` | Consecutive failures exceeded `DegradeThreshold`, or CRD missing from cluster |

Recovery from `degraded` to `healthy` happens on the next successful reconcile — no hysteresis.

## What it tracks

- **Reconcile counts** — total, failed, consecutive failures, last error, last reconcile time
- **Worker states** — per-worker idle/processing/stopped, plus aggregate counters; updated atomically on each reconcile item, reflected in Prometheus gauges immediately
- **Dependency status** — kept fresh by `dependencyHealthChecker`; flows into `/katalog/{crd}` and the Control Center
- **Autoscaler snapshot** — `workerInfoFn` and `autoMetricsFn` closures set by `wireCRDHealthCallbacks` during startup; called on every `/katalog/{crd}` request for a live snapshot; omitted when no autoscaler is configured
- **Rollback tracking** — callbacks injected via `SetRollbackNotifiers`; increments on trigger, clears on new spec generation *(rollback in development)*

## RuntimeHealth

`RuntimeHealth` is the operator-level aggregate. `/health` reflects it. `/ready` reflects it plus whether `Kordinate()` has started. It transitions to degraded when any CRD is missing or degraded, and recovers when all CRDs are started.

---

**Next →** [03 — Startup](03-startup.md)
