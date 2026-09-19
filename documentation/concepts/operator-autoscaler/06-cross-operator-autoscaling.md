# Cross‑Operator Autoscaling  
*How operators scale based on each other’s load.*

Cross‑Operator Autoscaling allows an operatorBox: to scale based on the **runtime metrics of another operator**.  
This enables upstream/downstream coordination, pipeline‑wide optimization, and ecosystem‑level behavior — all expressed declaratively in YAML.

This feature is powered by Orkestra’s cross‑operator IPC layer, which exposes:

- `.spec`  
- `.status`  
- `.labels`  
- **`.metrics` (live runtime metrics)**  

…from any referenced operator.

Cross‑operator metrics are read in‑memory (same binary) or via HTTP fallback (cross‑binary / cross‑cluster).  
No API calls, no polling, no external systems.

---

## Why Cross‑Operator Autoscaling Matters

Default OperatorBox autoscaling in Orkestra only considers **local** metrics:

- queue depth  
- worker utilization  
- reconcile latency  
- error rate  

But real systems are pipelines:

```text
ingest → transform → validate → store → index → notify
```

If a downstream operator is overwhelmed, upstream operators should slow down.  
If a downstream operator is idle, upstream operators can accelerate.

Cross‑operator autoscaling makes this possible.

---

## How It Works

When a CRD declares an `observe.cross:` block:

```yaml
operatorBox:
  observe:
    cross:
      - crd: database
        selector:
          name: "{{ .metadata.name }}-db"
        as: db
```

The autoscaler automatically receives:

```text
cross.db.metrics.queueDepth
cross.db.metrics.workersBusyPercent
cross.db.metrics.workersIdlePercent
cross.db.metrics.reconcileDurationP95Ms
cross.db.metrics.errorRatePercent
```

These values are injected into the condition engine exactly like local metrics.

---

## Supported Cross‑Operator Metric Fields

| Field | Description |
|---|---|
| `cross.<alias>.metrics.queueDepth` | Queue depth of the referenced operator |
| `cross.<alias>.metrics.workersBusyPercent` | Busy worker percentage |
| `cross.<alias>.metrics.workersIdlePercent` | Idle worker percentage |
| `cross.<alias>.metrics.reconcileDurationP95Ms` | P95 reconcile duration |
| `cross.<alias>.metrics.errorRatePercent` | Error rate |

If the referenced operator is not found, the metrics block is omitted and all cross‑metric conditions evaluate to **false**.

---

## Example: Scale Based on Downstream Pressure

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    when:
      - field: cross.db.metrics.queueDepth
        greaterThan: "500"
      - field: cross.db.metrics.workersBusyPercent
        greaterThan: "70"

  do:
    workers: 12
    queueDepth: 1500
```

**Behavior:**

- The operator scales up only when the **database operator** is under load  
- Prevents upstream overload  
- Enables pipeline‑wide stability  

---

## Example: Slow Down When Downstream Is Saturated

```yaml
autoscale:
  interval: 10s
  cooldown: 30s

  conditions:
    when:
      - field: cross.transformer.metrics.workersBusyPercent
        greaterThan: "90"

  do:
    workers: 2
    queueDepth: 50
```

**Behavior:**

- If the transformer operator is overwhelmed, the upstream operator slows down  
- Prevents cascading failures  
- Reduces backpressure and queue explosions  

---

## Example: Multi‑Operator Coordination

```yaml
autoscale:
  interval: 30s
  cooldown: 2m

  conditions:
    when:
      - field: cross.ingest.metrics.queueDepth
        greaterThan: "1000"
      - field: cross.storage.metrics.workersIdlePercent
        greaterThan: "40"

  do:
    workers: 20
```

**Behavior:**

- Scale up only when:  
  - ingest is overloaded  
  - storage has capacity  
- This creates a **balanced pipeline**  

---

## Example: Cross‑Operator + Local Metrics

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    or:
      - field: cross.db.metrics.errorRatePercent
        greaterThan: "5"

    when:
      - field: metrics.queueDepth
        greaterThan: "300"

  do:
    workers: 10
```

**Behavior:**

- If the database operator is failing too often **OR**  
- If this operator is under load  
- Then scale up  

This blends local and cross‑operator signals.

---

## Runtime Behavior

Cross‑operator metrics are resolved through:

1. **Informer cache** (same binary)  
2. **HTTP fallback** (cross‑binary / cross‑cluster)  
3. **Not‑found map** (if neither path is available)  

Metrics are injected into the autoscaler’s evaluation context on every tick.

All evaluations are:

- in‑memory  
- O(1)  
- lock‑free  
- zero API calls  

---

## Safety Guarantees

Cross‑operator autoscaling is designed to be safe:

- No circular dependencies (cross metrics are read‑only)  
- No deadlocks (operators never wait on each other)  
- No flapping (cooldown applies normally)  
- No dropped queue items  
- No goroutine leaks  

Operators remain fully isolated inside their operatorBox:es.

---

## When to Use Cross‑Operator Autoscaling

Use it when:

- you have upstream/downstream relationships  
- you have pipelines or DAGs  
- you want to prevent overload propagation  
- you want to coordinate multiple operators  
- you want ecosystem‑level optimization  

Avoid it when:

- operators are unrelated  
- the downstream operator is ephemeral  
- the dependency graph is unclear  

---

## Summary

Cross‑Operator Autoscaling enables:

- pipeline‑wide scaling  
- upstream/downstream coordination  
- distributed load awareness  
- ecosystem‑level optimization  
- fully declarative behavior  

This is one of Orkestra’s most powerful capabilities — and a major differentiator from traditional operator frameworks.
