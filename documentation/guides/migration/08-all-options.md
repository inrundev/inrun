# All Options

All five migration patterns running in a single Orkestra runtime. One binary, one Komposer, five CRD kinds.

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/08-all-options
make registry KOMPOSER=komposer-local.yaml && make build
ork simulate
ork run -f komposer-local.yaml
```

---

## What is here

Each option lives in `options/` as a self-contained sub-directory. They share one `go.mod` and compile into one binary.

| Option | CRD Kind | Pattern |
|--------|----------|---------|
| `declarative` | `DeclarativeApp` | Pure YAML, zero Go |
| `hybrid` | `HybridApp` | Declarative + Go service hook |
| `hooks` | `HooksApp` | Go hook, no declared templates |
| `constructor` | `ConstructorApp` | Go reconcile loop (direct migration) |
| `ork-resources` | `OrkApp` | Go reconcile loop (Orkestra resources) |

All five produce the same result: a Deployment and a Service for each CR. The difference is how much Go is involved in producing them.

---

## How the Komposer composes them

`komposer-local.yaml` imports each option's `katalog.yaml` from `options/`. Each katalog declares its own unique group and kind, so all five coexist in the same runtime without conflict.

The Komposer adds two things at composition level:

**Dependency ordering.** `hybridApp` and `hooksApp` wait for `declarativeApp` to start before their reconcilers activate:

```yaml
spec:
  crds:
    hybridApp:
      dependsOn:
        declarativeApp:
          condition: started
    hooksApp:
      dependsOn:
        declarativeApp:
          condition: started
```

**Worker tuning.** The Komposer overrides `workers` for specific CRDs where the katalog's defaults are too conservative for the combined load.

That is all a Komposer adds here — the katalogs are self-contained. The same approach scales to production: import from OCI references instead of local files, tune at the Komposer level, keep the katalogs untouched.

---

## Observing side-by-side

With all five running:

```bash
kubectl get deployments
kubectl get services
kubectl get declarativeapps,hybridapps,hooksapps,constructorapps,orkapps
```

Open the Control Center to watch each reconciler's queue depth, worker utilisation, and health state independently:

```bash
ork control
```

`declarativeApp` activates first. `hybridApp` and `hooksApp` become active once it is started. `ConstructorApp` and `OrkApp` run independently from the start.

---

## OCI distribution

Once each option validates locally, publish it so others can pull the Komposer without cloning the repo:

```bash
# Repeat for each option you want to distribute
ork push -f options/<name>/katalog.yaml
```

Update `komposer.yaml` with your registry org and tags, then:

```bash
ork pull -f komposer.yaml
make registry && make build
ork run
```

The OCI imports replace the local file imports — same runtime, distributed katalogs.

---

## When to use this

Use this as a template for real multi-operator runtimes — one Komposer, multiple katalogs from different teams. The per-katalog isolation means each team owns their CRD entry; the Komposer owner controls load balancing, dependency order, and distribution.
