# 03 — Hooks Only

Option three. Go owns both resources — Deployment and Service — in a single hook. There are no declared templates in the Katalog.

`pkg/resources` (`orkdeploy.Update`, `orksvc.Update`) handles create-if-absent, drift correction, owner references, and system labels in one call per resource — replacing the manual Get / IsNotFound / Create / Patch from the baseline.

The runtime still provides the informer, workqueue, worker pool, finalizers, events, metrics, and status.

---

> **Before you start:** Update [katalog.yaml](katalog.yaml) — set `author:` to your name or org and `metadata.name` to your operator name. Replace `ghcr.io/myorg` throughout this guide with your own image registry and org. Set your katalog registry before pushing:
> ```bash
> export ORK_REGISTRY=ghcr.io/myorg/katalogs
> ```

---

## What changed from 02-hybrid

| | 02-hybrid | 03-hooks-only |
|---|---|---|
| Deployment | `operatorBox.onCreate.deployments` — declared | Go hook |
| Service | Go hook (`hooks/webapp_hooks.go`) | Go hook |
| Declared templates | Yes (Deployment) | None |
| Go required | Yes | Yes |
| Reconcile order | hook first, then declared templates | hook only |

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

Both the Deployment and the Service appear in cycle 1.

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
export IMAGE_REPO=ghcr.io/myorg/webapp-hooks-only
export IMAGE_TAG=1.0.0
make release
```

`make release` compiles with the `runtime` build tag (no validate/simulate/e2e commands), builds the distroless image, and pushes it.

## Step 5. Update [values.yaml](values.yaml) with your image

The e2e gate runs automatically during push and needs to pull your custom runtime image. Update [values.yaml](values.yaml) to point to the image you just built:

```yaml
runtime:
  image:
    repository: ghcr.io/myorg/webapp-hooks-only
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
ork inspect webapp-hooks-only:1.0.0
```

Expected:

```text
webapp-hooks-only:1.0.0
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

[04 — Constructor migration](../04-constructor-migration/README.md) — lift the full `Reconcile()` into Orkestra.
