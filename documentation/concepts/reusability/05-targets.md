# Targets — One CRD, Multiple Reconcile Strategies

In the `mixed-operator pattern` example pack, a single runtime runs three different CRDs — each with its own operatorBox. One CRD is purely declarative. One uses a typed hooks binary. One uses a constructor. The Komposer composes them and they run together.

Targets are the same idea, applied to a single CRD.

Instead of three CRDs with three different operatorBoxes, you have one CRD with three named surfaces — and each surface has its own operatorBox. Each target defines what gets created, how reconciliation works, and under what conditions it runs. The schema is shared. The gateway surface is shared. What varies is declared per target.

!!! tip "Try the Mixed Operator pattern"
      ```bash
      ork init --pack advanced/11-mixed-operator-pattern
      ```
      Follow the steps in the README.

---

## Each target has its own operatorBox

A target's operatorBox works exactly like a CRD-level operatorBox. It can be declarative — creating resources without any Go code. It can declare hooks. It can declare a constructor. It can import motifs. It can define preReconcile gates that control when reconciliation runs.

```yaml
serve:
  target:
    standard:
      operatorBox:
        reconcile:
          onCreate:
            deployments:
              - image: "{{ .spec.image }}"   # declarative — no hooks needed
    managed:
      operatorBox:
        reconcile:
          hooks:
            location: github.com/myorg/my-operator/hooks
            function: ManagedHooks
            alias: managed
            args:
              tier: managed
        preReconcile:
          enqueueGate:
            when:
              - field: '{{ inBusinessHours }}'
                equals: "true"
    custom:
      operatorBox:
        reconcile:
          default: false
          constructor:
            location: github.com/myorg/my-operator/reconciler
            function: NewCustomReconciler
            alias: custom
```

A caller picks a surface. The runtime applies that surface's operatorBox — its gates, its hooks or constructor, its declared resources — for every CR that arrives from that surface.

---

## What a target's operatorBox can declare

Anything a CRD-level operatorBox can declare is available per target:

| Declaration | What it does |
|-------------|-------------|
| `onCreate` resources | Declaratively create Deployments, Services, ConfigMaps, etc. when the CR is created |
| `reconcile.hooks` | A typed hook binary that runs on each reconcile event |
| `reconcile.constructor` | A custom reconciler that owns the full reconcile loop |
| `preReconcile.enqueueGate` | A condition that must be true before a CR is enqueued |
| `preReconcile.reconcileGate` | A condition that must be true before reconciliation proceeds |
| `imports` | One or more Motifs, composing shared behaviour into the target |
| `args` | Configuration values passed to hooks or the constructor |
| `status.fields` | Status fields written after each reconcile cycle |

The CRD-level operatorBox is still the base. A target's operatorBox extends or overrides it for that surface.

---

## Targets in a Katalog

```yaml
spec:
  crds:
    blockchainapp:
      apiTypes: ...

      operatorBox:          # base — applies to all targets unless overridden
        reconcile:
          workers: 2
          resync: 30s
      serve:
        target:
          v2-enabled:
            primary: true
            operatorBox:
              reconcile:
                hooks:
                  location: github.com/myorg/blockchain/hooks
                  function: BlockchainHooks
                  alias: bchooks
                  args:
                    featureEnabled: "true"
              preReconcile:
                enqueueGate:
                  when:
                    - field: '{{ inBusinessHours }}'
                      equals: "true"
          v2-disabled:
            operatorBox:
              reconcile:
                hooks:
                  location: github.com/myorg/blockchain/hooks
                  function: BlockchainHooks
                  alias: bchooks
                  args:
                    featureEnabled: "false"
          v2-custom:
            operatorBox:
              reconcile:
                default: false
                constructor:
                  location: github.com/myorg/blockchain/reconciler
                  function: NewBlockchainReconciler
                  alias: bcctor
```

Three surfaces. One CRD. The same schema, the same informer, the same gateway — with different operatorBoxes behind each surface.

When a CR moves from one surface to another, the runtime cleans up what the previous surface created before the new surface takes over.

---

## Related topics

- [OperatorBox](../operatorbox/index.md) — the full operatorBox schema: onCreate, reconciler, preReconcile, status, and imports
- [Reconciler Model](../reconciler-model/index.md) — how a CR moves through the reconcile loop from enqueue to status patch
- [Args](04-args.md) — configuration-only variation across targets without changing reconcile logic
