# 01 — ResourceKatalog

`ResourceKatalog` is the per-GVK registry. It answers the question: "what do I need to start reconciling this CRD?"

Each GVK maps to a `RegistryEntry` holding the full Katalog config for that CRD, the running informer, a factory that builds a fresh reconciler on demand, and the degrade threshold.

The registry is written exactly once during `konstructRuntime`, before `DependencyKordinator` starts. After that it is read-only.

**Three callers read from it at runtime:**
- `startCRDWorkers` — calls `ReconcilerFactory()` to produce the reconciler
- `/katalog/{crd}` handler — reads static CRD config for the response
- Worker loop — reads `entry.Informer.GetIndexer().List()` to update resource count metrics

---

**Next →** [02 — CRDHealth](02-health.md)
