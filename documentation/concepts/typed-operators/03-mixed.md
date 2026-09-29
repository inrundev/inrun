# Mixing Declarative, Hooks, and Constructor

A single Orkestra runtime can run all three operator patterns together. Each CRD in the Katalog is independent — one can be purely declarative, another can use hooks, a third can use a constructor. They share the same process, the same informer factory, the same health server.

---

## Example

To try this example:

```bash
ork init --pack advanced
cd 11-mixed-operator-pattern
ork run
```

The example composes three Katalogs into one runtime:

```yaml
# komposer.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: mixed-operator-pattern

imports:
  files:
    - ./01-hello-website/katalog.yaml   # pure declarative — no Go
    - ./09-hooks/katalog.yaml           # typed hooks (Database)
    - ./10-constructor/katalog.yaml     # typed constructor (Pipeline)

spec:
  crds:
    database:
      operatorBox:
        reconcile:
          workers: 5
    website:
      dependsOn:
        database:
          condition: healthy
    pipeline:
      dependsOn:
        website:
          condition: started
```

The three Katalogs:

- **Declarative** — `01-hello-website/katalog.yaml`: creates Deployments and Services from templates, no Go
- **Hooks** — `09-hooks/katalog.yaml`: typed Go hooks for the Database CRD, declarative status layer still applies
- **Constructor** — `10-constructor/katalog.yaml`: custom reconciler for the Pipeline CRD, state machine pattern

---

## Generate and build

One command handles the complete mixed project:

```bash
ork generate registry --file komposer.yaml
go build ./cmd/orkestra
```

`ork generate registry` scans the Komposer, resolves all three Katalogs, and emits:

- `pkg/typeregistry/zz_generated_typeregistry.go` — registers all typed CRDs (Database, Pipeline) with the Kubernetes scheme
- `cmd/orkestra/main.go` — the operator entry point

The declarative Website CRD needs no Go types — it is included in the registry automatically. The generated files cover the whole runtime.
