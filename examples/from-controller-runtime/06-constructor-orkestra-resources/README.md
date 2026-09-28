# 05 — Constructor: Orkestra Resources

Option 04 lifted the reconcile logic unchanged. This option goes one step further: the Get → IsNotFound → Create → Patch pattern inside the sub-methods is replaced by `pkg/resources`. The constructor signature and struct stay identical — only the resource management calls change.

`orkdeploy.Update`, `orksvc.Update` each handle: create if absent, patch if drifted, owner references, and system labels — in one call per resource.

---

> **Before you start:** Update [katalog.yaml](katalog.yaml) — set `author:` to your name or org and `metadata.name` to your operator name. Replace `ghcr.io/myorg` throughout this guide with your own image registry and org. Set your katalog registry before pushing:
> ```bash
> export ORK_REGISTRY=ghcr.io/myorg/katalogs
> ```

---

## What changed from 04

| | 04-constructor-migration | 05-constructor-orkestra-resources |
|---|---|---|
| Reconcile signature | `Reconcile(ctx, key string) error` | unchanged |
| Struct | `WebAppReconciler{kube, scheme}` | unchanged |
| Deployment management | Get → IsNotFound → Create → Patch | `orkdeploy.Update` |
| Service management | Get → IsNotFound → Create → Patch | `orksvc.Update` |
| Owner references | manual | automatic |
| Drift correction | manual patch | automatic |
| Lines of Go | ~120 | ~60 |

---

## Step 1 — Generate the type registry

```bash
make registry
```

Generates `pkg/typeregistry/zz_generated_typeregistry.go` from your Katalog. Re-run whenever `apiTypes` changes.

---

## Step 2 — Build

```bash
make build
```

Builds the full CLI (validate, simulate, run) and places it at `~/.orkestra/bin/ork`.

---

## Step 3 — Validate

```bash
make validate
```

---

## Step 4 — Simulate

```bash
ork simulate
```

---

## Step 5 — Run locally

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

## Step 6. Build and push the production image

```bash
export IMAGE_REPO=ghcr.io/myorg/webapp-constructor-ork
export IMAGE_TAG=1.0.0
make release
```

`make release` compiles with the `runtime` build tag (no validate/simulate/e2e commands), builds the distroless image, and pushes it.

## Step 6. Update [values.yaml](values.yaml) with your image

The e2e gate runs automatically during push and needs to pull your custom runtime image. Update [values.yaml](values.yaml) to point to the image you just built:

```yaml
runtime:
  image:
    repository: ghcr.io/myorg/webapp-constructor-ork
    tag: 1.0.0
```

## Step 7. Push the katalog pattern to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

> **Note:** `ork push` requires `docker login` to any OCI-compatible registry.

Simulate and e2e run automatically before the artifact and its dependencies are published.

**8. Confirm the published artifact**

```bash
ork inspect webapp-constructor-ork:1.0.0
```

Expected:

```text
webapp-constructor-ork:1.0.0
  ...
  Simulate:    ✓ Verified · 4 assertions · 227ms · tested 44s ago
  E2E:         ✓ Verified · 2 assertions · 20s · tested 18s ago
  Typed:       ✓ constructor · requires custom runtime image
  Runtime:     v0.7.7
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[06 — ork migrate](../06-ork-migrate/README.md) — automate the option 04 path for your own operator.
