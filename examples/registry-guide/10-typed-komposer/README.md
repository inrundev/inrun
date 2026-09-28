# 11 — Typed Komposer

Combine three katalogs — two declarative and one typed — into a single Orkestra Runtime binary. One process watches and reconciles three CRDs: WebApp, Cache, and Database. All three operators are pulled from the registry; no local files.

---

## Files

| File | Purpose |
|------|---------|
| `komposer.yaml` | Imports webapp + cache + database katalogs from the registry |
| `simulate.yaml` | Aggregator: runs simulate for all three operators |
| `cr-webapp.yaml` | Sample WebApp CR |
| `cr-cache.yaml` | Sample Cache CR |
| `cr-database.yaml` | Sample Database CR |

---

> **Before you start:** Replace `ghcr.io/myorg` with your actual registry path throughout this example. All three katalogs must have been published — see [02](../02-katalog-api/README.md), [03](../03-katalog-cache/README.md), and [10](../10-hooks-katalog/README.md).

---

## Runtime binary

Because `komposer.yaml` imports `database-operator` (a typed katalog with Go hooks), validate and simulate require a binary compiled with the type registry.

**If you just completed [10-hooks-katalog](../10-hooks-katalog/README.md):** the binary is already at `~/.orkestra/bin/ork`. Skip this step.

**Starting fresh:**

```bash
cd ../10-hooks-katalog
make build
cd -
```

`make build` generates the type registry and compiles a full CLI binary with the Database hooks registered. Output: `~/.orkestra/bin/ork`.

---

## Pull

Pull all three katalogs to local cache in one command:

```bash
ork pull -f komposer.yaml
```

All OCI imports resolve and cache at `~/.orkestra/registry/`. Validate, simulate, and template all run from cache — no network needed after this step.

---

## Validate

```bash
ork validate
```

---

## Simulate

Run all three operators in one command:

```bash
ork simulate
```

`simulate.yaml` is an aggregator — it imports the individual `simulate.yaml` from each operator example and runs them all. No cluster needed.

---

## Build and push the runtime image

**Already done in [10-hooks-katalog](../10-hooks-katalog/README.md)?** Skip this — the image is already in your registry.

**First time:**

```bash
cd ../10-hooks-katalog
make release IMAGE_REPO=ghcr.io/myorg/database-operator IMAGE_TAG=v1.0.0
cd -
```

`make release` compiles the production binary (with the `runtime` build tag), builds the distroless image, and pushes it.

---

## Deploy in-cluster

Apply the CRDs from the cached artifacts (already pulled above):

```bash
ork inspect webapp-operator:v2.0.0   --view crd.yaml | kubectl apply -f -
ork inspect cache-operator:v1.0.0    --view crd.yaml | kubectl apply -f -
ork inspect database-operator:v1.0.0 --view crd.yaml | kubectl apply -f -
```

Generate and apply the bundle:

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

Install Orkestra with the typed runtime image:

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set runtime.image.repository=ghcr.io/myorg/database-operator \
  --set runtime.image.tag=v1.0.0 \
  --wait --timeout 120s
```

Apply instances of any or all CRDs:

```bash
kubectl apply -f cr-webapp.yaml
kubectl apply -f cr-cache.yaml
kubectl apply -f cr-database.yaml
```

---

## Mixing declarative and typed

The Komposer treats declarative and typed katalogs identically at the import level:

```yaml
imports:
  registry:
    - oci://ghcr.io/myorg/katalogs/webapp-operator:v2.0.0         # v1.0.0 is deprecated
    - oci://ghcr.io/myorg/katalogs/cache-operator:v1.0.0
    - oci://ghcr.io/myorg/katalogs/database-operator:v1.0.0
```

The difference is in the binary. When you run `make build` in `10-hooks-katalog/`, the generated type registry includes registrations for the Database type and hook factory. automatically. Declarative katalogs compile into the standard Generic Reconciler with no extra code.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next step

→ [12-ork-action](../12-ork-action/README.md) — automate the entire pipeline with GitHub Actions
