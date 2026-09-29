# 04 — Worker Info

## Purpose

`WorkerInfo` is the serialisable snapshot of an operatorbox's worker state. It is returned as `autoscalerWorkers` in the `/katalog/{crd}` response and rendered in the Control Center.

Before `WorkerInfo`, the endpoint showed a raw goroutine count that was inflated to `max(baseline, override)` at startup. Workers looked "pending" even when idle. `WorkerInfo` replaces this with a precise, real-time picture.

## Fields

```go
type WorkerInfo struct {
    Configured           int     // baseline declared in Katalog
    Effective            int     // current semaphore capacity (may be overridden)
    InFlight             int     // goroutines inside reconcile right now
    Idle                 int     // Effective - InFlight
    Max                  int     // highest override value from do.workers (omitted if no autoscaler)
    AutoscalerEnabled    bool    // true when autoscale: is declared
    OverrideActive       bool    // true when override is currently applied
    OverrideWorkers      int     // current override value (omitted when not active)
    QueueDepth           int64   // live workqueue length
    QueueDepthConfigured int     // baseline queue depth
    QueueDepthEffective  int     // current queue depth limit (baseline or override)
    BusyPercent          float64 // InFlight / Effective × 100
}
```

## Building a WorkerInfo

`BuildWorkerInfo` is called on every `/katalog/{crd}` request. It reads from the live semaphore and `AutoMetrics` — all O(1) atomic reads, no locks beyond the semaphore mutex:

```go
info := autoscaler.BuildWorkerInfo(
    r.workerSem,
    r.autoMetrics,
    configuredWorkers,
    configuredQueueDepth,
    maxWorkers,
    autoscalerEnabled,
    r.autoscaler.Snapshot(),
)
```

`Snapshot()` captures the autoscaler's current state (override active, effective workers/queueDepth) without holding the autoscaler lock.

## How it reaches the Control Center

```
/katalog/{crd} handler
    │
    ├── h.GetWorkerInfo()          // calls workerInfoFn() → WorkerInfo
    │       workerInfoFn set by kordinator after reconciler construction
    │
    └── CRDInfoResponse{
            AutoscalerWorkers: *WorkerInfo,   // omitempty — nil when no autoscaler
        }
```

The Control Center renders this as a clickable autoscaler panel on the CRD detail page, shown only when `AutoscalerWorkers` is non-nil. The panel expands to show baseline, `do:` block, and condition configuration.

## Goroutine model

Workers are started at the declared `baseline.workers` count. When the autoscaler calls `ResizeWorkers(n)` with `n > old`, `kordinator.ResizeWorkers` calls the injected `spawnWorker` callback once per new slot, starting additional goroutines on demand. This keeps the initial goroutine count at the baseline and avoids pre-allocating goroutines for a scale-up that may never occur.
