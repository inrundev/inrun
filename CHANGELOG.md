# Changelog

Inrun follows [Semantic Versioning](https://semver.org/). Until `v1.0`, minor
versions may contain breaking changes.

## Unreleased

Inrun is the intent runner that began as Orkestra. This is its first version
under the new name, refocused on one loop: an intent comes in through the
gateway, becomes a custom resource, is reconciled, and returns as a view.

### Changed

- **New name and identity.** Resources created by Orkestra are not adopted.

  | What | Now |
  | --- | --- |
  | Go module | `github.com/inrundev/inrun` |
  | CLI | `inrun` |
  | Console | `inrun-console` |
  | Images | `ghcr.io/inrundev/inrun`, `inrun-gateway`, `inrun-console` |
  | Helm chart | `inrun` |
  | API group, labels, annotations, finalizers | `inrun.dev` |
  | Environment variables | `INRUN_*` |

- **Kinds renamed.** The schema is unchanged.
  - `Katalog` → `Catalog` (`catalog.yaml`)
  - `Komposer` → `Stack` (`stack.yaml`)
  - `Motif` → `Module` (`module.yaml`)
- **`inrun` runs your project.** With no subcommand it runs the Catalog or
  Stack in the current directory; `inrun <name>:<version>` runs a published
  pattern. `inrun run` is gone.
- **One project layout.** `catalog.yaml` at the root, CRD, CR and setup files
  in `manifests/`, and `simulate.yaml`, `e2e.yaml` and test values in `test/`.
  `inrun simulate` and `inrun e2e` look in `test/` by default; `inrun init`,
  `create pattern` and `migrate` write this layout.
- **Examples** are four packs: declarative, remote, typed and intent.
- **The Control Center is now the console** (`inrun console`).
- **Helm chart** starts at `0.1.0`. The unused inline Catalog values are
  gone; point `catalog.existingConfigMap` at the ConfigMap from
  `inrun generate bundle`.

### Added

- `inrun simulate` (and `--envtest`) runs remote reconcilers against their
  endpoint.
- `inrun generate bundle`, `configmap` and `rbac` accept Catalogs that use
  `crdFile`; the Catalog they embed has `apiTypes` filled in and no local
  paths.

### Fixed

- e2e mutations (`patch`, `scale`, …) run once per expectation instead of on
  every retry.
- `inrun e2e` teardown on an existing cluster no longer hangs on a custom
  resource whose controller is already gone.
- Helm chart: HPAs target the right Deployments,
  `gateway.catalog.existingConfigMap` is honoured, and server timeouts are set
  when leader election is off.
- Failed steps show ✗ in CI logs.

### Removed

- Exploratory features that don't serve the intent loop.

## Earlier versions

Inrun's history before the rename is Orkestra's: see the `archive` branch and
the `orkestra-v0.7.18` tag.
