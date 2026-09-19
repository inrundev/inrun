# Hooks Only

All three resources — ServiceAccount, Deployment, and Service — are created in Go. There are no declared templates in the Katalog alongside the hook. Go owns the full child resource spec.

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/03-hooks-only
```

---

## What you will learn

- How to move all resource management into a hook
- How `pkg/resources` (`orkdeploy.Update`, `orksvc.Update`, `orksa.Update`) replaces the Get → IsNotFound → Create → Patch pattern
- What Orkestra still provides even when Go owns all resources — informer, workqueue, metrics, leader election, deletion lifecycle, admission serving

---

## What changed from hybrid

The Katalog declares the hook and resource kinds for RBAC, but no `onCreate` templates. The hook creates everything:

```yaml
operatorBox:
  reconcile:
    hooks:
      location: github.com/myorg/webapp-operator/hooks
      function: WebAppHooks
      managedResources:
        - kind: ServiceAccount
        - kind: Deployment
        - kind: Service
```

`pkg/resources` handles create-if-absent, drift correction, owner references, and system labels in one call per resource. Compare with the baseline's manual Get / IsNotFound / Create / Patch — same outcome, a fraction of the code.

---

## When to pick this over hybrid

When every resource requires computed logic that would feel artificial to declare, or when keeping declarations alongside a hook that owns everything would split the logic without benefit. If any resource is straightforward to declare, hybrid is usually the cleaner choice.

---

## Try it

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/03-hooks-only
# Follow steps in README
```

→ [04 — Constructor: lift and change](./05-constructor.md)
