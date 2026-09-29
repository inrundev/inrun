# reconciler.requeue

`reconcile.requeue` schedules a re-enqueue of the CR after a successful reconcile. The CR is added back to the workqueue after `after:` elapses — no informer event is needed.

Failed reconciles use `queue.retryBackoff`, not `requeue:`.

---

## Declaration

```yaml
spec:
  crds:
    myapp:
      operatorBox:
        reconcile:
          requeue:
            after: '{{ .spec.checkInterval | default "60s" }}'
            when:
              - field: status.phase
                notEquals: "Complete"
```

---

## Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `after` | duration or template | — | How long to wait before re-enqueuing. A Go duration string (`"30s"`, `"5m"`, `"1h"`) or a template expression evaluated against the live CR. Empty string disables requeue. |
| `when` | `[]Condition` | — | AND conditions. Requeue only fires when all pass. Omit to requeue unconditionally after every successful reconcile. |
| `or` | `[]Condition` | — | OR conditions. When both `when:` and `or:` are present, both must pass. |

### `after:` as a template

The expression is evaluated against the live CR at reconcile time using the full template context (fields, notes, profiles). Each CR can carry its own timing:

```yaml
after: '{{ .spec.syncInterval | default "2m" }}'
```

If the expression renders to an empty string, `"0s"`, or fails to parse as a duration, no requeue is scheduled.

### Validation

`ork validate` rejects a non-template `after:` value that is not a valid Go duration:

```text
✗ crd "myapp": requeue.after "every-minute" is not a valid duration or template expression.
  Use a Go duration string (e.g. "30s", "5m") or a template (e.g. '{{ .spec.interval | default "60s" }}')
```

---

## Relationship to other timing fields

| Field | Error path | Success path | Scope |
|-------|-----------|--------------|-------|
| `queue.retryBackoff` | ✓ | — | Per CRD |
| `reconcile.resync` | — | ✓ (uniform) | Per CRD, all CRs |
| `reconcile.requeue` | — | ✓ (conditional) | Per CR, template-driven |

`requeue:` and `resync:` are additive — whichever fires first re-enqueues the CR.

---

## See also

- [Requeue concept](../../../concepts/reconciler-model/03-requeue.md)
- [queue](14-queue.md) — `retryBackoff` and queue depth
