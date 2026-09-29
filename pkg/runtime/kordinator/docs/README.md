# Kordinator — Developer Documentation

## Documents

| File | Covers |
|---|---|
| [01-registry.md](01-registry.md) | `ResourceKatalog` — the per-GVK registry that holds every informer and reconciler factory |
| [02-health.md](02-health.md) | `CRDHealth` — health states, worker tracking, dependency status, autoscaler snapshot |
| [03-startup.md](03-startup.md) | Startup sequence — dependency channels, non-blocking activation, `startCRDWorkers` |
| [04-self-healing.md](04-self-healing.md) | The retry loop and its four phases — missing CRDs, runtime deletion, reappearance, deferred activation |
<<<<<<< HEAD
| [05-workers.md](05-workers.md) | Worker loop, `processItemForGVK` phases, autoscale resizing, drain |
| [06-normalize.md](06-normalize.md) | `normalize:` spec normalization — pipeline position, declare in Katalog, note functions |
| [07-health-reporting.md](07-health-reporting.md) | Multi-replica health: `isKonductor` flag, leader vs follower endpoint responses |
=======
| [05-workers.md](05-workers.md) | The worker loop, `processItemForGVK`, queue drain, and shutdown semantics |
| [06-handlers.md](06-handlers.md) | The three runtime introspection HTTP handlers that power the Control Center |

## Multi-replica health reporting

Running the runtime with `replicaCount > 1` introduces a split-brain health state problem — only the leader reconciles, but all pods serve traffic. The `health-reporting/` subfolder documents the full solution end-to-end.

| File | Covers |
|---|---|
| [health-reporting/01-overview.md](health-reporting/01-overview.md) | The problem, the `isKonductor` signal, how it flows |
| [health-reporting/02-runtime.md](health-reporting/02-runtime.md) | `RuntimeHealth.isKonductor`, `Kordinate()` lifecycle, real JSON responses from leader and follower |
| [health-reporting/03-control-center.md](health-reporting/03-control-center.md) | Connection pooling root cause, cache-update guard, CRD detail retry logic |
| [health-reporting/04-diagnosis.md](health-reporting/04-diagnosis.md) | Diagnosing flapping health, port-forward inspection commands, common failure patterns |
>>>>>>> origin/main
