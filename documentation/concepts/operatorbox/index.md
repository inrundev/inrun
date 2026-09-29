# OperatorBox

_CRDs in. Operators out._

You write a CRD declaration. Orkestra wraps it in an operatorBox and runs it as a fully isolated operator — its own informer, its own queue, its own worker pool, its own health state. No Go code required unless you want it.

Think of it the way you think of containers. A container takes an image and runs it in isolation from everything else on the host. An operatorBox takes a CRD declaration and runs it in isolation from every other CRD in the same process. Panics don't escape. Queue pressure doesn't spill. Health state is tracked independently.

The operatorBox is what makes Orkestra a **runtime** rather than a framework.

---

## What an operatorBox owns

Every operatorBox owns exclusively:

**Informer** — A `SharedIndexInformer` watching exactly the resources declared by this CRD entry. The in-memory cache holds all CR instances. Reads are zero-cost.

**Event queue** — A bounded workqueue receiving events from the informer. Queue depth and rate limiting are configured per CRD via `workers:` and `queue:` settings.

**Worker pool** — A fixed number of goroutines pulling from the queue. Configured via `workers:` in the Katalog.

**Reconciler** — A `Generic Reconciler[T]` holding the full CRD entry configuration: normalize rules, mutation rules, validation rules, template declarations, and cross-CRD informer references.

**Health state** — A `CRDHealth` instance tracking success rate, consecutive failures, and degraded threshold. Visible in the Control Center per operatorBox.

**Metrics** — Reconcile duration, error rate, queue depth, worker utilization — all labeled by GVK and scoped to this operatorBox.

**Lifecycle** — Starts and stops independently. Starts in dependency order. Drains queue on shutdown.

---

## Declaring an operatorBox

In the Katalog, the `operatorBox:` block declares the complete behavior of one CRD's isolated runtime:

```yaml
spec:
  crds:
    pipeline:
      crdFile: my-pipeline.yaml
      dependsOn:
        database: healthy

      normalize:
        spec:
          schedule: "{{ ... }}"

      operatorBox:
        reconcile:
          workers: 10
          resync: 30s
          onCreate: [...]        # resource creation on first reconcile
          onReconcile: [...]     # drift correction on every reconcile
          onDelete: [...]        # cleanup before finalizer removal
        emit:
          status: [...]          # status field declarations
        # reconcile: is optional — omit for declarative-only CRDs (Generic Reconciler is the default)
        # reconcile:
        #   default: false     # set to use a custom constructor instead
        #   constructor:
        #     location: ...
        observe:
          cross: [...]         # cross-CRD observations
```

Omitting `reconcile:` uses the declarative Generic Reconciler — no Go code required. Add `reconcile.constructor:` with `default: false` when you need a typed Go reconciler. Add `reconcile.remote:` with `default: false` when the reconcile logic lives in a separate HTTP service — any language, no Kubernetes SDK required.

---

## Where to go next

- [Reconcile Pipeline](01-reconcile-pipeline/index.md) — the ordered steps from queue to status patch, drift semantics, error behavior
- [Normalize](04-normalize/index.md) — accept multiple input shapes, produce one canonical spec
- [Enrich](05-enrich/index.md) — fetch live child state and embed it in template context
- [Profiles](06-profiles/index.md) — named presets for resources, security, probes, rollout, and PDB
- [External](07-external/index.md) — HTTP calls before resource reconciliation: health gates, config injection, image signing
- [Isolation and IPC](02-isolation.md) — how isolation is enforced and how operatorBoxes communicate
- [Startup Sequencing](03-startup-sequencing.md) — dependency order and the Katalog relationship
