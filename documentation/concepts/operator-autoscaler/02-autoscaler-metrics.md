# Autoscaler Metrics
*The metrics namespace used by the Operator Autoscaler.*

The Operator Autoscaler evaluates conditions using a dedicated metrics namespace:  
`metrics.*`.

These metrics are collected **inside the Orkestra runtime**, per CRD, inside each operatorBox.  
They are **in‑memory**, **O(1)** to read, and require **no API calls**, **no informers**, and **no Prometheus**.

Metrics are updated continuously by:

- the worker pool  
- the workqueue  
- the reconcile loop  

They represent the **true runtime load** of an operator.

---

## Metric fields

The autoscaler supports the following metric fields:

| Field | Type | Description |
|---|---|---|
| `metrics.workersBusyPercent` | float | Percentage of workers currently reconciling |
| `metrics.workersIdlePercent` | float | Percentage of workers waiting for work |
| `metrics.queueDepth` | int | Current number of items in the operator’s queue |
| `metrics.reconcileDurationP95Ms` | float | P95 reconcile duration (ms) over the last window |
| `metrics.errorRatePercent` | float | Percentage of reconciles that failed in the last window |

These fields are validated at Katalog load time.  
Unknown fields cause a **fail‑fast** error.

---

## How metrics are collected

### 1. Worker utilization

The worker pool tracks:

- total workers  
- workers currently executing `Reconcile`  
- workers blocked on the semaphore  

From this, Orkestra computes:

```text
workersBusyPercent = (busy / total) * 100
workersIdlePercent = (idle / total) * 100
```

These values update on every reconcile start/finish.

---

### 2. Queue depth

The workqueue exposes its current length:

```text
queueDepth = queue.Len()
```

This is read directly from the in‑memory queue structure.

---

### 3. Reconcile duration (P95)

Each reconcile records its duration into a ring buffer.  
Every autoscaler tick, Orkestra computes:

```text
P95 = 95th percentile of durations in the last N seconds
```

This is a rolling window, not a cumulative metric.

---

### 4. Error rate

The autoscaler maintains counters:

- total reconciles  
- failed reconciles  

Over the last window:

```text
errorRatePercent = (failed / total) * 100
```

This is reset periodically to avoid unbounded growth.

---

## Metric resolution

When the autoscaler evaluates a condition like:

```yaml
field: metrics.queueDepth
greaterThan: "500"
```

The condition engine:

1. Detects the `metrics.` prefix  
2. Routes the lookup to the autoscaler metrics subsystem  
3. Retrieves the current in‑memory value  
4. Compares it using the declared operator (`greaterThan`, `lessThan`, etc.)

This lookup is **constant‑time** and **never touches the API server**.

---

## Validation

At Katalog load time, Orkestra validates:

- all metric field names  
- all comparison operators  
- all numeric values  

Invalid metric fields produce an error like:

```text
unknown autoscale metric field "metrics.queueDpth" — valid fields:
metrics.workersBusyPercent, metrics.workersIdlePercent,
metrics.queueDepth, metrics.reconcileDurationP95Ms,
metrics.errorRatePercent
```

This ensures autoscaler conditions are always resolvable at runtime.

---

## Examples

### Scale when queue is deep and workers are saturated

```yaml
when:
  - field: metrics.queueDepth
    greaterThan: "300"
  - field: metrics.workersBusyPercent
    greaterThan: "80"
```

### Scale down when workers are mostly idle

```yaml
when:
  - field: metrics.workersIdlePercent
    greaterThan: "60"
```

### Scale when reconciles are slow

```yaml
when:
  - field: metrics.reconcileDurationP95Ms
    greaterThan: "250"
```

### Scale when error rate spikes

```yaml
when:
  - field: metrics.errorRatePercent
    greaterThan: "10"
```

---

## Reading the signals

| Metric | What it measures |
|---|---|
| Queue depth | Backlog — how much work is waiting |
| Worker utilization | Saturation — are all workers busy |
| Reconcile duration | Latency — how long each reconcile takes |
| Error rate | Instability — how often reconciles fail |

These metrics describe operator load directly, without needing external Prometheus rules or Pod-level resource metrics.
