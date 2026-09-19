# Kordinator

The kordinator is the part of the Orkestra runtime that manages when each CRD starts reconciling, how many workers it runs, and what happens when things change at runtime.

---

## Startup in dependency order

When your Katalog declares multiple CRDs, the kordinator resolves their `dependsOn:` declarations into a startup sequence. CRDs without dependencies start first. A CRD that depends on another waits until that dependency has reached the declared condition before its workers start.

```yaml
spec:
  crds:
    database:
      dependsOn: {}          # starts first

    api-server:
      dependsOn:
        database:
          condition: healthy  # waits until database has reconciled at least once
```

Two conditions are available:

| Condition | Meaning |
|-----------|---------|
| `started` (default) | The dependency's workers are running |
| `healthy` | The dependency has completed at least one successful reconcile |

If a dependency is not yet ready when the operator starts, the kordinator skips that CRD and a background loop checks periodically, starting it as soon as the condition is met. Nothing blocks the rest of the operator.

---

## Per-CRD worker pools

Each CRD gets its own independent worker pool. Workers run concurrently — the number is set by `reconcile.workers:` and defaults to 1.

```yaml
operatorBox:
  reconcile:
    workers: 3
```

Three workers means three CRs of that type can be reconciling simultaneously. Workers for one CRD do not share a pool with any other CRD.

Worker count can be adjusted at runtime without restarting the operator. The `autoscale:` block lets the kordinator scale workers up or down based on queue depth, reconcile latency, or custom conditions — see [Operator Autoscaler](../operator-autoscaler/).

---

## Health states

The kordinator tracks a health state for each CRD as it processes items:

| State | Meaning |
|-------|---------|
| `pending` | Workers not yet started — dependency conditions not met |
| `started` | Workers running, no successful reconcile yet |
| `healthy` | At least one successful reconcile completed |
| `degraded` | Consecutive failures exceeded the threshold, or the CRD has disappeared from the cluster |

These states feed the Control Center dashboard and the `/katalog` health endpoints. A dependent CRD waiting for `condition: healthy` unblocks the moment its dependency transitions to healthy — no restart required.

---

## Self-healing

The kordinator monitors running CRDs throughout the operator's lifetime. If a CRD is deleted from the cluster after the operator started, the kordinator stops that CRD's workers, marks it degraded, and propagates the change to any dependents. When the CRD reappears, workers restart automatically without restarting the operator.

This means an operator started before its CRDs are installed in the cluster will come up healthy once the CRDs are applied — no ordering requirement on installation.

---

## Per-target reconciliation

When a Katalog declares `serve.target:` entries, each CR carries a target annotation set by the gateway at delivery time. The kordinator routes each reconcile to the correct operatorBox for that target — different workers, different hooks, different args — all within the same CRD's worker pool.

See [serve targets](../self-service/02-target-mode.md) for how targets are declared.

---

## Relationship to the reconciler

The kordinator and the reconciler are separate concerns. The kordinator decides when to call `Reconcile` and how many concurrent calls to allow. The reconciler decides what to do with a specific CR. Neither depends on the other's implementation — the kordinator calls any `domain.Reconciler`, whether it is the GenericReconciler or your own constructor.
