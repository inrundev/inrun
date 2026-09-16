# 05 — Workers and drain

Each worker goroutine runs `runWorkerForGVK` for the lifetime of its CRD's context. Every CRD gets its own queue — a slow CRD cannot starve workers for a fast one.

## The worker loop

The loop dequeues an item, marks the worker processing, calls `processItemForGVK`, then marks it idle. After each item it updates queue depth and resource count metrics. It exits when the CRD context is cancelled or the queue shuts down.

## processItemForGVK

Handles one reconcile item end-to-end:

1. **Pre-reconcile gate** — evaluates `operatorBox.preReconcile.when/or` conditions. When gated, the item is forgotten (not a failure; health state unaffected).
2. **`prepare.Prepare`** — normalize, resolver, cross-CRD enrichment, mutation, validation. Returns `nil` when the object is not in cache (deleted between dequeue and here) — silent skip.
3. **`maintain.Apply`** — applies system labels, annotations, and finalizers.
4. **`safeReconcile`** — calls `rec.Reconcile(ctx, domain.Request{...})`. Any panic is caught, logged with a stack trace, and converted into a reconcile error. The worker goroutine continues.
5. **`post.Apply`** — status patch, event emission, runtime annotations. Runs even on reconcile failure.

On error the item is re-queued with rate-limit backoff. On success it is forgotten.

## Autoscale: worker resizing

When `autoscale:` is declared, `startCRDWorkers` starts only the baseline worker count. The autoscaler calls `kordinatorTarget.ResizeWorkers(n)` to resize the worker pool; this calls the `spawnWorker` closure in `perCRDRuntime` for each new slot. After each item the worker pushes live queue depth into `rt.autoMetrics` so autoscale conditions evaluate against current state.

## stopCRDWorkers

Cancel CRD context → shut down the queue → wait for workers with a drain timeout. The queue shutdown is required: a worker blocked on `Get()` waiting for the next item does not observe context cancellation; the shutdown unblocks it.

---

**Next →** [06 — normalize](06-normalize.md)
