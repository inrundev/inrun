# Retry backoff

When a reconcile call or an external data fetch fails, Orkestra re-enqueues the item. By default, re-enqueue happens via the workqueue's built-in rate limiter with no additional waiting. `retryBackoff` lets you layer *intra-reconcile* retries on top — so a transient failure is retried a configurable number of times with exponential backoff before the error is returned to the queue.

There are two places to declare it:

| Declaration site | What it retries |
|---|---|
| `queue.retryBackoff` | Each call to the reconciler function when it returns an error |
| `external[].retryBackoff` | That specific external call before its error propagates to the reconciler |

---

## Shorthand vs full form

Both sites accept the same two forms:

```yaml
# Shorthand — initial delay only; Orkestra supplies max: 30s, multiplier: 2.0, maxAttempts: 3
queue:
  retryBackoff: 5s

# Full form — explicit control over every parameter
queue:
  retryBackoff:
    initial: 500ms
    max: 30s
    multiplier: 2.0
    maxAttempts: 3
```

The shorthand `5s` is equivalent to `initial: 5s` with defaults applied to all other fields.

---

## `queue.retryBackoff` — reconciler retries

Wraps each call to your reconciler. If the reconciler returns an error, Orkestra waits `initial`, doubles the delay (up to `max`), and retries — up to `maxAttempts` times before returning the error to the workqueue.

```yaml
operatorBox:
  reconcile:
    resync: 10m
    queue:
      retryBackoff:
        initial: 500ms
        max: 30s
        multiplier: 2.0
        maxAttempts: 3
```

With the above, a failing reconcile is attempted 3 times with delays of 500ms and 1s (total ≈ 1.5s) before the error is returned and the item is re-enqueued by the workqueue's rate limiter.

### Resync interaction

If `initial`, `max`, and `maxAttempts` are set such that the worst-case retry window exceeds the `resync` period, `ork validate` emits a **warning** (not an error). The warning surfaces the math:

```text
queue.retryBackoff worst-case delay (150s) exceeds resync (30s) — the queue will
re-enqueue before retries finish; consider reducing maxAttempts or initial delay
```

This is a warning because it is not always wrong — a slow external API might justify deep in-call retries. The operator author decides.

---

## `external[].retryBackoff` — per-call retries

Retries a specific external call before its error reaches the reconciler. Use this when one call is known to be flaky without affecting the rest of the reconcile pipeline:

```yaml
operatorBox:
  reconcile:
    onReconcile:
      external:
        - name: health-check
          url: "{{ .spec.serviceUrl }}/health"
          retryBackoff:
            initial: 1s
            max: 10s
            multiplier: 1.5
            maxAttempts: 3
        - name: db-query
          url: "postgres://{{ .spec.dbHost }}/mydb"
          query: "SELECT 1"
          retryBackoff: 2s   # shorthand — 2s initial, defaults for the rest
```

The retry happens *inside* the external call executor before the result is placed in `.external.<name>`. If all attempts fail and `continueOnError: false` (the default), the error is returned to the reconciler.

---

## Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `initial` | duration | `500ms` | First backoff delay |
| `max` | duration | `30s` | Upper cap — the delay never grows beyond this |
| `multiplier` | float | `2.0` | Factor applied to the delay after each attempt |
| `maxAttempts` | int | `3` | Total calls including the first (1 = no retries) |

Shorthand (plain duration string) sets `initial` only; all other fields use their defaults.

---
!!! tip "Not a substitute for idempotency"
    Retries assume your reconciler is idempotent — calling it twice produces the same result as calling it once. If your reconciler creates resources that are not cleaned up on failure, retrying it multiple times can create duplicates. Fix the idempotency first; then add retries.

## When to use each

**`queue.retryBackoff`** — use when the reconciler as a whole can be safely retried. Good for operators that call external services that are occasionally unavailable, and where partial completion is not a concern (all resource operations are idempotent).

**`external[].retryBackoff`** — use when one specific external call is the flaky part and you want to shield the rest of the reconcile from it. More targeted than retrying the whole reconciler.

---

## Where to go next

- [External calls](07-external/index.md) — full guide to `external:` declarations, result context, and patterns for health gating and config injection
- [Queue](../../reference/schema/02-katalog/14-queue.md) — `maxDepth`, `failureThreshold`, `shared`, and `retryBackoff` reference
- [Reconciler model](../reconciler-model/) — how items move from the informer cache through the workqueue to the reconciler
- [Health subsystem](../health-subsystem/) — how `failureThreshold` and degraded state interact with `dependsOn`

