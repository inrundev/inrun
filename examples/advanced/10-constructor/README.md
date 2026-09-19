# 10 — Custom Constructor

**When you need maximal control — migrate existing operators, run state machines, or own the reconcile loop.**

Orkestra is declarative first. Most operators — even complex ones — are fully expressible in a Katalog using `when:` conditions, dependencies, status fields, and hook templates. Only when you have an existing controller‑runtime operator that you want to bring in as‑is, or when you need complete control over the reconcile loop (e.g., a custom state machine not easily described in YAML), do you reach for a custom constructor.

---

## The three layers of Orkestra

| Layer | What you write | Use case |
|-------|----------------|----------|
| **Declarative Katalog** | YAML (status, when, resources) | 95% of operators |
| **Typed Hooks** | Go functions (`OnReconcile`, `OnDelete`) | Adding complex external calls, custom logic inside the safe loop |
| **Custom Constructor** | Full `domain.Reconciler` implementation | Migrating existing controllers, state machines that own the loop |

**Try declarative first. If you can’t, try hooks. Only then consider a constructor.**

---

## What a constructor gives you – and what you lose

| Feature | Declarative / Hooks | Constructor |
|---------|---------------------|--------------|
| Finalizer management | Automatic | You handle |
| Status (Ready condition) | Automatic | You write |
| Declared status fields | Written automatically | Ignored |
| Kubernetes events | Automatic | You emit |
| Drift correction (onReconcile templates) | Yes | You implement |
| Panic recovery | Yes | Yes (still safe) |
| Metrics (reconcile duration, errors) | Yes | Yes |
| Worker pool, queue, informer | Yes | Yes |

**Constructor gives you full control over the reconcile method. Use it when you already have an existing reconciler and want to plug it into Orkestra without rewriting.**

---

## How to migrate an existing controller‑runtime operator

Suppose you have a standard controller‑runtime reconciler:

```go
// Existing code – controller‑runtime
func (r *PipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // your logic here
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}
```

To move it to Orkestra:

1. Change the signature to `Reconcile(ctx context.Context, key string) error`
   - `key` is `namespace/name` (same as `req.String()`)
   - Return `nil` for done, `error` to requeue with exponential backoff
2. Remove the manager setup – Orkestra provides the informer, queue, and metrics.
3. Register your constructor in the Katalog with `reconcile.default: false`
4. Run `ork generate registry` to wire it into Orkestra.

That’s it. Your existing reconcile logic runs inside Orkestra’s runtime infrastructure.

---

## When you might still want a constructor (migration aside)

- **State machines with many phases** – you could also do this declaratively with `when:` conditions, but a constructor lets you centralise the state transition logic.
- **Complex external API dependencies** – if the hook model’s `OnReconcile` is too restrictive (e.g., you need to requeue conditionally based on external state), a constructor gives you full control.
- **Gradual adoption** – you have a large codebase that already implements `Reconcile`, and you want to run it under Orkestra while you later refactor parts into hooks.

Otherwise, **use declarative Katalog or typed hooks**.

---

## The example: Pipeline state machine

This example demonstrates a constructor that runs a series of Jobs (build → test → notify). It is intentionally written as a constructor to show the pattern. But **the same behaviour can be expressed declaratively** in the Katalog. We provide the constructor version for learning and migration reference.

---

## Files in this pack

```
.
├── api/v1alpha/           ← Pipeline CRD Go types
├── reconciler/            ← custom reconciler implementation
│   └── reconciler.go      ← NewPipelineReconciler + Reconcile logic
├── cmd/orkestra/          ← main.go (imports generated registry)
├── pkg/typeregistry/           ← generated registry (after `make registry`)
├── katalog.yaml
├── Makefile
├── Dockerfile
└── crd.yaml, cr.yaml
```

---

## Step 1 — Generate the registry and entrypoint

Run the registry generator:

```bash
make registry
```

This executes:

```bash
ork generate registry --file katalog.yaml
```

It creates (or updates) two files:

- `pkg/typeregistry/zz_generated_typeregistry.go` – registers your Go types and hooks.
- `cmd/orkestra/main.go` – the entrypoint that imports the generated registry.

Both files are marked `DO NOT EDIT` – they are regenerated whenever you change the Katalog.

---

## Step 2 – Build your custom binary

First, see the expected error with the standard `ork` CLI:

```bash
ork validate
# CRD "pipeline": no constructor registered — check reconciler.constructor in Katalog and re-run ork generate registry
```

Now build your own binary:

```bash
make clean
make build
```

This replaces the default `ork` binary in `~/.orkestra/bin/ork`, which is already on your PATH from the initial install.

Validate:

```bash
ork validate   # passes
```

---

## Step 3 – Run locally

```bash
ork run --dev     # creates a local kind cluster

kubectl apply -f crd.yaml
```

In another terminal:

```bash
kubectl apply -f cr.yaml
kubectl get pipeline -w
kubectl get jobs -w
```

You’ll see the state machine step through `build` → `test` → `notify`.

---

## Step 4 – Deploy to a cluster

```bash
# Generate Bundle
ork generate bundle -f katalog.yaml -o bundle.yaml

# Build and push your custom image
make release IMAGE=yourregistry/pipeline-operator:v1

# Apply bundle
kubectl apply -f bundle.yaml

# Deploy orkestra
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --set runtime.image.repository=yourregistry/pipeline-operator \
  --set runtime.image.tag=v1 \
  --namespace orkestra-system \
  --wait --timeout 120s

# Apply if not done already
kubectl apply -f crd.yaml
kubectl apply -f cr.yaml
```

---

## E2E

Typed operators require your own published image — `ork e2e` installs Orkestra via Helm and you must point it at your image so the deployed runtime includes your generated type registry.

**Step 1 — build and push your image:**

```bash
make docker push IMAGE_REPO=yourregistry/pipeline-operator IMAGE_TAG=latest
```


**Important notes: (build-time security)**

- `make docker` builds a production‑only binary (tags `runtime`) – it cannot run developer commands like `validate` or `e2e`
- The binary is copied from `~/.orkestra/bin/runtime/ork` into the current directory for `docker build`, then removed

#### Try running a developer command
It will fail (as intended), only `ork run` succeeds.

```bash
~/.orkestra/bin/runtime/ork validate
# unknown command "validate" for "ork"
```

---

**Step 2 — run e2e with your image:**

```bash
ork e2e \
  --set runtime.image.repository=yourregistry/pipeline-operator \
  --set runtime.image.tag=latest
```

The `--set` flags are are used by `ork e2e`, so the cluster runs your image instead of the default Orkestra image.

This spins up a kind cluster, deploys your custom Orkestra runtime, applies the CR, asserts the state machine ran to completion, then tears down.

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Pipeline deployment created
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        namespace: default
        ready: true

  - name: Deployment removed on delete
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        namespace: default
        count: 0
```

---

## Cleanup

```bash
helm uninstall orkestra -n orkestra-system
kubectl delete -f bundle.yaml
kind delete cluster --name orkestra-playground
```

---

## Next steps

- If you have an existing controller‑runtime operator, try migrating it using this pattern.
- For new operators, start with the declarative Katalog (example 01–08). Only reach for a constructor if you truly need to own the reconcile loop.