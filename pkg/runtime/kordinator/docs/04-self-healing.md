# 04 — Self-healing

The startup loop runs once. Everything that happens after — CRDs that arrive late, CRDs that are deleted and recreated, CRDs waiting for a `healthy` dependency — is handled by two goroutines launched before the startup loop begins:

- `retryMissingCRDs` — activation, deactivation, and deferred startup
- `dependencyHealthChecker` — runtime dependency health evaluation and `healthyCh` signalling

Both run for the operator's lifetime and stop only when the context is cancelled.

## retryMissingCRDs

Runs on `postStartRetryInterval` (90s in-cluster, 10s when running locally via `ork run`). The interval is selected at startup using `utils.IsRunningInPod()` — no configuration required. Each tick executes four phases in order.

### Phase 1 — Detect runtime disappearances

Iterates every registered GVK. For each one that is not already in the missing map, calls `crdExists()` to check whether the CRD still exists in the cluster. When a CRD has vanished:

```
entry.Missing = true
crdHealthMap[gvk].SetMissingAtRuntime()
crdHealthMap[gvk].SetDegraded()
orkHealth.SetKatalogDegraded()
deactivateCRD(gvk)
```

Dependents that require `condition: healthy` on the missing CRD are also marked degraded. Dependents that only require `condition: started` are left running — their `startedCh` was already closed and they can continue processing their own CRs.

### Phase 2 — Re-activate missing CRDs

Iterates the missing map and calls `crdExists()` for each entry. When a CRD reappears:

```
activateCRD(ctx, entry)
deactivated[gvk] = false
crdHealthMap[gvk].SetStarted()
```

When CRDs remain missing, exponential backoff increases the sleep between `crdExists()` calls to avoid hammering the API server.

### Phase 3 — Deferred activation

Iterates the full topological order looking for CRDs that were skipped during the startup loop. For each CRD that is not yet started and not currently missing:

```go
if k.dependenciesReady(crd, nameToGVK) {
    activateCRD(ctx, entry)
}
```

This is the path that activates CRDs waiting for `condition: healthy`. Once `dependencyHealthChecker` closes `healthyCh[depGVK]`, the next tick of Phase 3 finds `dependenciesReady()` returning true and activates the waiting CRD.

### Phase 4 — Aggregate health

```go
if len(k.informerFactory.Missing()) == 0 && k.allCRDsStarted() {
    k.allOnline.Store(true)
    k.orkHealth.SetKatalogReady()
}
```

## activateCRD

`activateCRD` is the single path for bringing a CRD online, whether from Phase 2 or Phase 3:

1. Starts the informer if it was never started (`entry.WasNeverStarted`)
2. Removes the GVK from the missing map
3. Calls `startCRDWorkers`
4. Closes `startedCh[gvk]` using `select/default` to guard against double-close:

```go
select {
case <-k.startedCh[gvk]: // already closed — no-op
default:
    close(k.startedCh[gvk])
}
```

5. For dependents requiring `condition: healthy`, checks if the dependency health checker has already made the CRD healthy — if so, closes `healthyCh[gvk]` too

The `select/default` guard is essential. When a CRD is deleted and recreated, `activateCRD` runs twice. The first run closes the channel; the second run must not panic.

## deactivateCRD

`deactivateCRD` drains a running CRD's workers without closing `startedCh`:

1. Cancels the CRD's context — workers observe `ctx.Done()` and stop accepting new items
2. Adds drain sentinel items to the queue — one per worker — to unblock any goroutine currently waiting on `wq.Queue.Get()`
3. Waits for the WaitGroup with a timeout
4. Calls `ResetWorkerCounts()` and marks all workers as stopped
5. Sets `deactivated[gvk] = true`

`startedCh` is deliberately not closed. If it were closed, dependents would see a closed channel and interpret the CRD as ready — but the CRD's workers are stopped and it is no longer reconciling anything. Keeping the channel open means dependents remain in their current health state and resume normally when `activateCRD` is called again.

## dependencyHealthChecker

Runs on `RuntimeHealthCheckInterval`. For every active CRD, checks the health of each declared dependency and updates the `DependencyStatus` map in `CRDHealth`.

The critical path is the `healthyCh` signal:

```go
if depHealth.IsHealthy() {
    if !depHealth.SignaledHealthy() {
        close(k.healthyCh[depGVK])
        depHealth.MarkHealthySignaled()
    }
}
```

`healthyCh[depGVK]` is closed exactly once — the first time `IsHealthy()` returns true. `SignaledHealthy` / `MarkHealthySignaled` use an atomic flag to prevent a double-close race if two goroutines evaluate the condition simultaneously.

This is the integration point between runtime health and startup gating. The retry loop's Phase 3 polls `dependenciesReady()` which checks `healthyCh`. The health checker closes that channel. The two loops never coordinate directly — the channel is the only shared state between them.

## Scenarios at a glance

| Scenario | Phase that handles it |
|---|---|
| CRD missing at startup | Phase 2 activates when CRD appears |
| CRD deleted while running | Phase 1 detects and deactivates |
| CRD recreated after deletion | Phase 2 activates again |
| CRD waiting for `condition: started` | Phase 3 activates when `startedCh` closes |
| CRD waiting for `condition: healthy` | Phase 3 activates after health checker closes `healthyCh` |
| All CRDs online | Phase 4 sets katalog ready |

---

**Next →** [05 — Workers and drain](05-workers.md)
