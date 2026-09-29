# Multi-replica health reporting

With `replicaCount > 1` and leader election enabled, only the konductor pod runs reconcilers. Follower pods serve HTTP but their in-memory `CRDHealth` never advances — they always return `state: pending, healthy: false`.

## The `isKonductor` flag

The konductor pod sets an atomic flag on `RuntimeHealth` at the start of `Kordinate()`. Every health-sensitive endpoint includes it so consumers can tell whether the response came from an authoritative source.

```
Kordinate() called on winning pod
  → orkHealth.SetIsKonductor(true)
  → CRDHealth advances: pending → started → healthy

Pod loses leadership
  → orkHealth.SetIsKonductor(false)
```

Follower pods never call `Kordinate()` — their `isKonductor` stays `false` for their entire lifetime.

## What to expect per endpoint

`GET /katalog`:

| Field | Leader | Follower |
|-------|--------|----------|
| `isKonductor` | `true` | `false` |
| `healthy` | `true` | `false` |

`GET /katalog/{crd}/health`:

| Field | Leader | Follower |
|-------|--------|----------|
| `isKonductor` | `true` | `false` |
| `state` | `healthy` | `pending` |
| `totalReconciles` | > 0 | `0` |

`GET /katalog/{crd}`:

| Field | Leader | Follower |
|-------|--------|----------|
| `isKonductor` | `true` | `false` |
| `workersActive` | > 0 | `0` |
| `workerDetails` | populated | empty |

`resourceCount` is the same on both pods — it comes from the informer cache, synced from the Kubernetes API independently of leadership.
