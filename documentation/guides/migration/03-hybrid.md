# Hybrid

The Deployment and ServiceAccount are declared in the Katalog. The Service is created in a Go hook with access to the fully-typed CR. Orkestra runs the declared templates first, then calls the hook — or the other way around if `runHooksFirst: true`.

This is the 90/10 pattern. Most operators have some resources that are straightforward to declare and one or two that need Go. Hybrid lets you draw that line yourself.

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/02-hybrid
```

---

## What you will learn

- How to mix declared templates and a Go hook in one Katalog
- How `runHooksFirst` controls execution order
- When to use a hook versus `pkg/resources` in the declarative path
- How notes, conditionals, normalize, and cross-operator data can push the declared side further before you need Go

---

## What changed from declarative

The ServiceAccount and Deployment stay declared in the Katalog. The Service moves into Go:

```yaml
operatorBox:
  reconcile:
    hooks:
      location: github.com/myorg/webapp-operator/hooks
      function: WebAppHooks
      managedResources:
        - kind: Service
```

The hook receives `obj *apiv1.WebApp` — the fully-typed CR. It creates the Service using `pkg/resources`, which handles drift detection, owner references, and system labels automatically.

---

## Execution order

| `runHooksFirst` | Order |
|----------------|-------|
| `false` (default) | Declared templates → hook |
| `true` | Hook → declared templates |

See [Typed Operators — hooks](../../concepts/typed-operators/) for the full lifecycle.

---

## When to reach for the hook

Processes that cannot be expressed as a sequence of declarative steps. Before adding a hook, check whether [Notes](../../concepts/), [Conditionals](../../concepts/conditional/), [Normalize](../../concepts/operatorbox/04-normalize/), [External](../../concepts/operatorbox/07-external/), or [OnCOP](../../concepts/oncop/) covers the need. The declarative surface keeps expanding.

---

## Try it

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/02-hybrid
# Follow steps in README
```

→ [03 — Hooks only](./04-hooks-only.md)
