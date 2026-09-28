# 07 — All Options

All five WebApp migration patterns running in a single Orkestra runtime. Each option lives in `options/` as a self-contained sub-directory sharing one `go.mod`. A Komposer unifies them.

| Option | CRD Kind | Pattern |
|--------|----------|---------|
| `declarative` | `DeclarativeApp` | Pure YAML, zero Go |
| `hybrid` | `HybridApp` | Declarative + Go service hook |
| `hooks` | `HooksApp` | Go hook, no declared templates |
| `constructor` | `ConstructorApp` | Go reconcile loop (direct migration) |
| `ork-resources` | `OrkApp` | Go reconcile loop (Orkestra resources) |

`hybridApp` and `hooksApp` depend on `declarativeApp` — they wait for it to start before activating.

---

## How the Komposer works

`komposer-local.yaml` imports each option's `katalog.yaml` from `options/`. Each katalog declares its own unique group (`allopt.demo.orkestra.io`) and kind directly, so all five coexist in the same runtime without any apiTypes overrides.

The komposer adds only composition-level config — `workers` for the declarative, constructor, and ork-resources CRDs, and `dependsOn` so `hybridApp` and `hooksApp` wait for `declarativeApp` to start:

```yaml
spec:
  crds:
    hybridApp:
      dependsOn:
        declarativeApp:
          condition: started
```

The Go source files and hooks live under `options/<name>/` and are compiled into one binary via the shared root `go.mod`.

---

## Local run

### Step 1 — Generate registry and build

```bash
make registry KOMPOSER=komposer-local.yaml
make build
```

Generates a single type registry from all five local katalogs and compiles one binary.

### Step 2 — Validate

```bash
ork validate -f komposer-local.yaml
```

### Step 3 — Simulate

```bash
ork simulate
```

Runs all five options in one pass — no cluster needed.

### Step 4 — Run

Apply CRDs:

```bash
kubectl apply -f options/declarative/crd.yaml -f options/hybrid/crd.yaml \
  -f options/hooks/crd.yaml -f options/constructor/crd.yaml \
  -f options/ork-resources/crd.yaml
```

Start the runtime:

```bash
ork run -f komposer-local.yaml
```

In a second terminal, apply CRs:

```bash
kubectl apply -f options/declarative/cr.yaml -f options/hybrid/cr.yaml \
  -f options/hooks/cr.yaml -f options/constructor/cr.yaml \
  -f options/ork-resources/cr.yaml
```

Open the Control Center:

```bash
ork control
```

`declarativeApp` activates first. `hybridApp` and `hooksApp` unblock once it is started. All five reconcile loops run independently — each with its own queue depth, worker utilization, and health state.

Compare what each option created:

```bash
kubectl get deployments
kubectl get services
kubectl get declarativeapps,hybridapps,hooksapps,constructorapps,orkapps
```

All five produce the same result: a Deployment and a Service. The difference is the degree of Go involvement.

### Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Distribution (OCI)

Once you are happy with each option locally, publish them so anyone can pull the Komposer without cloning the repo.

Using hybrid as the example — repeat for each option you want to distribute:

```bash
cd ../02-hybrid
git add .
git commit -m "feat: webapp hybrid operator"
git tag v1.0.0
git push origin main --tags
ork push -f katalog.yaml
```

Update [komposer.yaml](komposer.yaml) with your registry org and tags, then:

```bash
ork pull -f komposer.yaml
make registry
make build
ork run
```

The OCI imports replace the local file imports — same runtime, distributed katalogs. Make sure your GitHub Package registry is public so the OCI references are accessible to consumers.
