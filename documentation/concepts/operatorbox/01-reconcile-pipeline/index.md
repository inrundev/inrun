# Reconcile Pipeline

When a queue item is dequeued, the operatorBox reconcile pipeline runs in this order:

```text
informer cache → DeepCopy → normalize → mutation → validation
    → OnReconcile hook (Go) or runTemplateReconcile (declarative)
        → cross-CRD observation  (.cross.*)
        → external HTTP calls    (.external.*)
        → forEach expansion
        → onCreate resource groups
        → onReconcile resource groups
    → patchStatusWithChildren
```

Each step receives the output of the previous step. No step can see the output of a later one.

---

**Normalize** produces the canonical spec. Template expressions in the `normalize:` block resolve here — defaults are expanded, field formats are coerced — before any other step sees the data.

**Mutation** applies declared defaults. Fields that should have a value when none was provided.

**Validation** enforces constraints. A failure stops the pipeline and writes the error to `status.conditions[Ready]`. The CR is not requeued unless it changes.

**Cross-CRD observation** reads the current state of other CRDs declared in `cross:`. Results are available as `.cross.<name>.*` in all subsequent template expressions.

**External HTTP calls** run before any resource group. Results are available as `.external.<name>.*`. See [External](../07-external/index.md).

**forEach expansion** expands list or map fields into repeated resource declarations. Each expansion adds `.item` and optional `.index` to the template context.

**onCreate / onReconcile resource groups** are the heart of the declarative path. Each resource group evaluates `when:`/`or:` conditions, resolves template expressions, and dispatches creates or updates. See [Drift](01-drift.md) for the exact semantics of what gets corrected and what does not.

**patchStatusWithChildren** always runs last, even when the pipeline returned an error. It writes the `Ready` condition, reads live child resource state, and resolves declared `status.fields`. See [Status Management](../../status-management/index.md).

---

## Where to go next

- [Drift](01-drift.md) — what counts as drift, onCreate vs onReconcile, what gets corrected, condition semantics, once: and deletion behavior
- [Error Behavior](02-error-behavior.md) — failure recording, backoff, degraded state, panic recovery
- [Panic Recovery](03-panic-recovery.md) — how safeReconcile catches panics, what is logged, isolation guarantees, common causes
