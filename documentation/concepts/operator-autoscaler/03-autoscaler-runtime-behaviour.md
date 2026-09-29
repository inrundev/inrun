# Autoscaler Runtime Behavior
*How the autoscaler behaves at runtime.*

The autoscaler runs as part of every operatorBox: that declares an `autoscale:` block.  
It evaluates conditions, applies overrides, restores baselines, and exposes live metrics — all in‑memory, without API calls or external controllers.

Autoscaling is **safe**, **deterministic**, and **fully reversible**.

---

## 1. Evaluation loop

The autoscaler runs on a fixed interval:

```yaml
interval: 30s
```

On every tick:

1. Read local metrics (`metrics.*`)
2. Read cross‑operator metrics (`cross.<alias>.metrics.*`)
3. Evaluate `or` (OR)
4. Evaluate `when` (AND)
5. Combine results  

```text
final = or_passes AND when_passes
```

This evaluation is O(1) and entirely in‑memory.

---

## 2. Applying overrides

If `final == true`, the autoscaler applies the `do:` overrides immediately:

- `workers:` → resizes the worker semaphore  
- `queueDepth:` → updates the queue’s max depth  
- `resync:` → adjusts the resync interval  

Overrides take effect **without restarting** the operator or Orkestra.

Workers scale up instantly.  
Workers scale down gracefully (no goroutines are killed).

---

## 3. Restoring baseline

If `final == false` for the entire `cooldown:` period:

```yaml
cooldown: 2m
```

…the autoscaler restores the CRD’s declared baseline:

- baseline worker count  
- baseline queue depth  
- baseline resync interval  

Restoration is also immediate and safe.

If `cooldown:` is omitted, restoration happens on the next tick.

---

## 4. Local metrics (`metrics.*`)

Each operatorBox: maintains its own live metrics:

- queue depth  
- busy/idle worker percentage  
- reconcile P95 duration  
- error rate  
- total reconciles  

These are updated continuously by the worker pool and reconcile loop.

All metrics are atomic and read without locking.

---

## 5. Cross‑operator metrics (`cross.<alias>.metrics.*`)

When an operator declares:

```yaml
operatorBox:
  observe:
    cross:
      - crd: database
        selector:
          name: "{{ .metadata.name }}-db"
        as: db
```

…the autoscaler automatically receives:

```text
cross.db.metrics.queueDepth
cross.db.metrics.workersBusyPercent
cross.db.metrics.errorRatePercent
cross.db.metrics.reconcileDurationP95Ms
```

Cross metrics come from:

1. **Informer cache** (same‑binary operators)  
2. **HTTP fallback** (cross‑binary / cross‑cluster)  
3. **Not‑found map** (if neither path is available)

If the referenced operator is not found, all cross‑metric conditions evaluate to **false**.

---

## 6. Interaction with the reconcile loop

Autoscaling never interrupts reconciliation.

- Scaling **up** increases concurrency immediately  
- Scaling **down** waits for in‑flight reconciles to finish  
- Queue depth changes do not drop items  
- Resync interval changes take effect on the next cycle  

The reconcile loop remains fully deterministic.

---

## 7. Interaction with drift correction

Drift correction (`reconcile: true`) continues to run normally:

- Overrides do not disable drift correction  
- Drift correction respects the current worker count  
- Resync overrides accelerate or slow down drift correction frequency  

Autoscaling and drift correction are orthogonal.

---

## 8. Logging

The autoscaler logs:

- condition evaluations  
- override applications  
- baseline restorations  
- cross‑operator metric reads  
- informer vs HTTP path selection  

This makes autoscaling fully observable in production.

---

## 9. Safety guarantees

The autoscaler guarantees:

- no goroutine leaks  
- no dropped queue items  
- no race conditions  
- no flapping (thanks to `cooldown:`)  
- no cross‑operator deadlocks  
- no dependency cycles (cross metrics are read‑only)  

Autoscaling is designed to be safe even under heavy load.

---

## 10. Summary

| Property | Behavior |
|---|---|
| Declarative | Expressed entirely in YAML |
| In-process | No external controllers, no additional deployments |
| Immediate | Overrides apply on the next tick |
| Reversible | Baseline restored automatically after `cooldown:` |
| Cross-operator | Operators can scale based on each other's metrics |
| Zero API calls | All metrics are in-memory reads |