# Generated files

`ork migrate -o ./my-operator` writes five files. Fields marked `TODO(ork migrate):` could not be inferred from the source and need manual review.

## katalog.yaml

The constructor Katalog stub. `ork migrate` extracts what it can from `SetupWithManager`:

| Field | Source | Detected? |
|-------|--------|-----------|
| `kind` | `For(&pkg.Kind{})` struct name | ✓ |
| `object` / `objectList` | same struct name + `List` suffix | ✓ |
| `version` | import path last segment (`v1alpha1`, `v1`) | ✓ |
| `location` | full import path of the type package | ✓ |
| `alias` | import alias in source | ✓ |
| `managedResources:` | each `Owns(&pkg.Kind{})` call | ✓ |
| `watch:` | each `Watches(&pkg.Kind{}, …)` call | ✓ |
| `group` | not in source (CRD marker or YAML) | ✗ TODO |
| `plural` | not in source | ✗ TODO |

Example output for a reconciler with `For(&demov1alpha1.WebApp{})`, `Owns(&appsv1.Deployment{})`, `Watches(&demov1alpha1.Config{}, …)`:

```yaml
metadata:
  name: web-app-reconciler   # derived from receiver type

spec:
  crds:
    web-app:
      apiTypes:
        group: TODO           # TODO(ork migrate): your CRD group
        version: v1alpha1     # from import path
        kind: WebApp          # from For()
        plural: TODO          # TODO(ork migrate)
        object: WebApp        # from For()
        objectList: WebAppList
        location: github.com/example/webapp-operator/api/v1alpha1
        alias: demov1alpha1

      operatorBox:
        watch:
          - apiVersion: TODO  # TODO(ork migrate): verify (custom package)
            kind: Config
            on: [create, update, delete]

        reconcile:
          default: false

          constructor:
            location: github.com/example/webapp-operator/controller
            function: NewWebAppReconciler
            managedResources:
              - kind: Deployment
                apiVersion: apps/v1
```

Only `group` and `plural` remain as TODOs for standard k8s `Owns()` types.

## simulate.yaml

Simulation stub. Fill in the resource kinds your operator creates in cycle 1:

```yaml
spec:
  expect:
    ops:
      - cycle: 1
        verb: create
        resource: TODO  # e.g. deployments
```

Run `ork simulate` after filling this in — no cluster required.

## e2e.yaml

End-to-end test stub. Fill in CRD kind, CR name, and assertions:

```yaml
spec:
  expect:
    - name: Resources created
      after: cr-applied
      timeout: 90s
      resources:
        - kind: TODO   # e.g. Deployment
          name: TODO   # e.g. my-webapp
          namespace: default
          ready: true
```

Run `ork e2e` to execute against a real cluster (kind is created automatically).

## go.mod

Module file with Orkestra pinned to the CLI version that ran the migration:

```
module github.com/myorg/my-operator

go 1.22

require (
    github.com/orkspace/orkestra v0.8.1
    k8s.io/api v0.29.3
    k8s.io/apimachinery v0.29.3
    k8s.io/client-go v0.29.3
)
```

Run `go mod tidy` to resolve indirect dependencies.

## The rewritten reconciler

Same filename as the input. See [01-output.md](01-output.md) for what changed.

---

Next: [Limitations](03-limitations.md)
