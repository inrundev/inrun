# resync vs requeue

Both primitives schedule reconciles on a timer, but they operate at different scopes and serve different purposes.

---

## resync — uniform, whole-CRD

`reconcile.resync:` is the informer resync period. On the interval, Orkestra re-lists every CR of the CRD from the API server and re-queues all of them, regardless of their individual state. Every object gets the same cadence.

```yaml
operatorBox:
  reconcile:
    resync: 10m   # re-enqueue every CR of this CRD every 10 minutes
```

Think of it as a safety net: even if a watch event is missed, the informer catches up on the next resync. It is a blunt instrument — it does not care whether a specific CR needs attention.

---

## requeue — targeted, per-object

`reconcile.requeue:` schedules a re-enqueue for the specific CR that just reconciled. Each object can carry its own timing, derived from its own fields. Other CRs are not affected.

```yaml
operatorBox:
  reconcile:
    requeue:
      after: '{{ .spec.checkInterval | default "60s" }}'
```

One CR with `spec.checkInterval: 30s` and another with `spec.checkInterval: 5m` each run on their own schedule, independent of each other and of resync.

---

## They compose

Both can be declared at the same time. The CR is re-enqueued by whichever fires first.

```yaml
operatorBox:
  reconcile:
    resync: 10m
    requeue:
      after: '{{ .spec.checkInterval | default "60s" }}'
```

A CR with a 30-second `checkInterval` reconciles roughly every 30 seconds. The 10-minute resync also fires, but the CR was already reconciled recently — the queue deduplicates and the extra enqueue is a no-op. A CR with no `checkInterval` falls back to the 60-second default. Everything still gets the 10-minute resync as a floor.

---

## Decision guide

| I want to… | Use |
|---|---|
| Re-evaluate all CRs periodically as a safety net | `resync:` |
| React when a secondary resource changes | `watch:` |
| Check a specific CR on its own schedule | `requeue:` |
| Re-check based on a field in the CR (`spec.ttl`, `status.certExpiry`) | `requeue:` with a template `after:` |
| Retry after a failed reconcile | `queue.retryBackoff:` — not requeue |

`requeue:` only fires on success. Errors are always handled by `queue.retryBackoff`.

---

## Relationship to typed operators

A typed reconciler signals per-object requeue timing by returning a non-zero `domain.Result.RequeueAfter`. The declarative `requeue:` block and the Go `RequeueAfter` field use the same workqueue path — they are two surfaces for the same mechanism.
