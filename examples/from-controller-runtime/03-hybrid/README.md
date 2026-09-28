# 02 — Hybrid

Option two. Declarative and Go working together in the same reconcile loop.

The Deployment is declared in the Katalog — Orkestra creates and drift-corrects it. The Service is created by a Go hook with type-safe access to `obj.Spec.Port`. Both run inside the same Orkestra reconcile cycle without you wiring anything together.

The hook runs *first* in this example — `runHooksFirst: true` in the Katalog. The Service is ready before Orkestra applies the declared templates. Omit `runHooksFirst` or set it to `false` to flip the order: declared templates run first, hook is additive after.

```yaml
hooks:
  runHooksFirst: true   # hook → then declared templates
                        # false (default): declared templates → then hook
```

The runtime still provides the informer, workqueue, worker pool, finalizers, events, metrics, and status.

---

> **Before you start:** Update [katalog.yaml](katalog.yaml) — set `author:` to your name or org and `metadata.name` to your operator name. Replace `ghcr.io/myorg` throughout this guide with your own image registry and org. Set your katalog registry before pushing:
> ```bash
> export ORK_REGISTRY=ghcr.io/myorg/katalogs
> ```

---

## What changed from 01-declarative

| | 01-declarative | 02-hybrid |
|---|---|---|
| Deployment | `operatorBox.onCreate.deployments` | unchanged — still declared |
| Service | `operatorBox.onCreate.services` | Go hook (`hooks/webapp_hooks.go`) |
| Go required | No | Yes — hook only |
| Custom runtime | No | Yes — `make build` |
| Reconcile order | runtime | hook first, then declared templates |

---

## Step 1 — Build the custom runtime

Generate the type registry and entrypoint:

```bash
make registry
```

Build your operator binary:

```bash
make build
```

This replaces the default `ork` binary in `~/.orkestra/bin/ork`, which is already on your PATH from the initial install.

---

## Step 2 — Simulate

```bash
ork simulate
```

Both the Deployment (declarative) and the Service (hook) appear in cycle 1.

---

## Step 3 — Run locally

Apply the CRD:

```bash
kubectl apply -f crd.yaml
```

```bash
ork run
```

In another terminal, apply the CR:

```bash
kubectl apply -f cr.yaml
kubectl get webapps
kubectl get deployments
kubectl get services
```

---

## Step 4 — Control center

In a third terminal:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) — both resources visible, hook activity in the reconcile log.

---

## Step 5. Build and push the production image

```bash
export IMAGE_REPO=ghcr.io/myorg/webapp-hooks
export IMAGE_TAG=1.0.0
make release
```

`make release` compiles with the `runtime` build tag (no validate/simulate/e2e commands), builds the distroless image, and pushes it.

## Step 5. Update [values.yaml](values.yaml) with your image

The e2e gate runs automatically during push and needs to pull your custom runtime image. Update [values.yaml](values.yaml) to point to the image you just built:

```yaml
runtime:
  image:
    repository: ghcr.io/myorg/webapp-hooks
    tag: 1.0.0
```

## Step 6. Push the katalog pattern to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

> **Note:** `ork push` requires `docker login` to any OCI-compatible registry.

Simulate and e2e run automatically before the artifact and its dependencies are published.

**7. Confirm the published artifact**

```bash
ork inspect webapp-hooks:1.0.0
```

Expected:

```text
webapp-hooks:1.0.0
  ...
  Simulate:    ✓ Verified · 4 assertions · 227ms · tested 44s ago
  E2E:         ✓ Verified · 2 assertions · 20s · tested 18s ago
  Typed:       ✓ hooks · requires custom runtime image
  Runtime:     v0.7.7
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[03 — hooks only](../03-hooks-only/README.md) — the hook owns all three resources; no declared templates.
