# Kordinator — Developer Documentation

## Documents

| File | Covers |
|---|---|
| [01-registry.md](01-registry.md) | `ResourceKatalog` — the per-GVK registry that holds every informer and reconciler factory |
| [02-health.md](02-health.md) | `CRDHealth` — health states, worker tracking, dependency status, autoscaler snapshot |
| [03-startup.md](03-startup.md) | Startup sequence — dependency channels, non-blocking activation, `startCRDWorkers` |
| [04-self-healing.md](04-self-healing.md) | The retry loop and its four phases — missing CRDs, runtime deletion, reappearance, deferred activation |
| [05-workers.md](05-workers.md) | Worker loop, `processItemForGVK` phases, autoscale resizing, drain |
| [06-normalize.md](06-normalize.md) | `normalize:` spec normalization — pipeline position, declare in Katalog, note functions |
| [07-health-reporting.md](07-health-reporting.md) | Multi-replica health: `isKonductor` flag, leader vs follower endpoint responses |
