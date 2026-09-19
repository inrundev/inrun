# Requeue

After a successful reconcile, Orkestra normally waits for the next informer event before running again. `reconcile.requeue:` changes this: declare a duration (static or template-driven) and Orkestra re-enqueues the CR on a timer — no external event required.

This is the right tool when your reconciler needs to act on time, not just on change.

---

## When to use it

`watch:` fires when another resource changes. `resync:` fires for all CRs uniformly. `requeue:` fires per-object, on a schedule derived from the object itself.

Use it when:

- Different objects need different polling intervals — `{{ .spec.checkInterval | default "60s" }}`
- You want to re-evaluate a condition on a timer — a lease expiry, a TTL, a certificate approaching rotation.
- A hook or external call produces a result that changes over time and you want to react without an external event source.

Do not use it as a substitute for `watch:`. If a resource your reconciler reads can send events, declare it in `watch:` instead — event-driven is always cheaper than polled.

---

## Basic usage

```yaml
operatorBox:
  reconcile:
    requeue:
      after: "60s"
```

After every successful reconcile, the CR is re-enqueued 60 seconds later. Failed reconciles are handled by `queue.retryBackoff` — `requeue:` only fires on success.

---

## Template expression

`after:` is evaluated against the live CR at reconcile time, so each CR can carry its own timing:

```yaml
operatorBox:
  reconcile:
    requeue:
      after: '{{ .spec.checkInterval | default "60s" }}'
```

If `.spec.checkInterval` is `"30s"` on one CR and `"5m"` on another, each gets its own schedule. The expression is re-evaluated every reconcile — changing `spec.checkInterval` on the CR takes effect on the next cycle.

---

## Conditional requeue

`when:` and `or:` gate the requeue. If the conditions fail, no requeue is scheduled — the CR waits for the next informer event instead.

```yaml
operatorBox:
  reconcile:
    requeue:
      after: "30s"
      when:
        - field: status.phase
          notEquals: "Complete"
```

The CR is re-enqueued every 30 seconds while `status.phase` is not `Complete`. Once it reaches `Complete`, requeue stops and the CR is idle until its next informer event.

Both `when:` (AND) and `or:` (OR) follow the same semantics as gate conditions. When both are present, both must pass.

---

## Relationship to other timing primitives

| Primitive | Fires when | Scope |
|-----------|-----------|-------|
| `requeue.after:` | After a successful reconcile, per-object timing | Per CR, template-driven |
| `reconcile.resync:` | On a fixed interval, for every CR of this CRD | Per CRD, uniform |
| `watch:` | When a declared secondary resource changes | Event-driven |
| `queue.retryBackoff:` | After a failed reconcile | Error path only |

`requeue:` and `resync:` are additive — both can be declared. The CR is re-enqueued by whichever fires first.

---

## Typed operators

Typed reconcilers (`domain.ReconcilerFrom`) set requeue timing by returning a non-zero `domain.Result.RequeueAfter`:

```go
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ...
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

`domain.ReconcilerFrom` forwards the value directly. The declarative `requeue:` block is for operatorBox reconcilers — both mechanisms use the same workqueue path.
