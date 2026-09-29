# pkg/runtime/reconcilers/genreic

`Generic Reconciler` is a client of Kordinator — one instance per CRD, called by the worker loop exactly like any other `domain.Reconciler`. It receives a fully-prepared `domain.Request` and owns the reconcile logic only: context enrichment, deletion routing, and declarative template dispatch.

Kordinator owns everything around it: worker pool, semaphore, autoscaler, prepare/maintain/post phases.

## Reconcile flow

```
Generic Reconciler.Reconcile(ctx, req)
  │
  ├── Context enrichment   logger, requestID, CRD name, resource key
  ├── Deletion check       → handleDeletion (hooks.OnDelete or runTemplateOnDelete)
  │
  └── reconcileImpl
        ├── Rollback gate  [in development] active rollback blocks normal reconcile;
        │                  runs onRollback templates until spec changes
        ├── Dispatch       hooks.OnReconcile  — typed Go callback
        │                  runTemplateReconcile — declarative onCreate/onReconcile
        │                  (no-op)            — kordinator still runs post.Apply
        ├── Rollback check [in development] on dispatch error: evaluate trigger conditions,
        │                  mark rollback active via RollbackGenerationAnnotation
        └── Spec snapshot  on success: write PreviousSpecAnnotation (gzip+base64),
                           clear failure history
```

Per-resource-type runners live in [`pkg/runtime/runners`](../../runners/README.md).

forEach expansion and child resource reading live in [`pkg/children`](../../../children/README.md).

The `pkg/resources/<kind>` package contract (Create, Apply, DeleteIfOwned, Resolve) is in [`pkg/resources`](../../../resources/README.md).
