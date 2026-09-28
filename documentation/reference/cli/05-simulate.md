# ork simulate

Run the operator reconcile loop against a fake in-memory cluster — no real Kubernetes cluster required. Useful for verifying that a declarative Katalog creates the right resources in the right order.

```bash
ork simulate
```

`--file` defaults to `simulate.yaml` (then `katalog.yaml`/`komposer.yaml`) and `--cr` defaults to `cr.yaml` — both are optional when your files use the standard names.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `simulate.yaml` | Path to simulate.yaml, katalog, or komposer file |
| `--cr` | | `cr.yaml` | Path to the CR YAML file to reconcile |
| `--crd` | | *(all)* | CRD name to simulate. Defaults to all CRDs in the Katalog. |
| `--cycles` | | `10` | Maximum number of reconcile cycles per CRD |
| `--skip` | | | Comma-separated path patterns to skip during `./...` discovery |
| `--skip-external` | | `false` | Stub `external:` HTTP calls with empty 200 responses |
| `--debug-ops` | | `false` | Print every recorded op with its cycle number |
| `--dev-server` | | `false` | Start the mock dev server for `external:` examples |
| `--dev-server-port` | | `9999` | Port for the mock dev server |
| `--envtest` | | `false` | Run against a local kube-apiserver + etcd instead of fake clients. Binaries are downloaded automatically to `~/.ork/envtest-bins` on first use. Requires `spec.crd` or `spec.crdFiles` in simulate.yaml. |
| `--k8s-version` | | `1.32` | Kubernetes version for envtest binaries (e.g. `1.31`, `1.32`). Two-component form only — `1.32.x` is not accepted. Only used with `--envtest`. |

## Examples

```bash
# Standard layout — no flags needed
ork simulate

# Non-standard filenames
ork simulate -f my-operator.yaml --cr my-cr.yaml

# Simulate a specific CRD only
ork simulate --crd website

# Run up to 5 reconcile cycles
ork simulate --cycles 5

# Discover and run all simulate.yaml files recursively
ork simulate ./...

# Start the mock dev server for external: examples
ork simulate --dev-server

# Run against a local kube-apiserver (real CRD enforcement, real watch streams)
ork simulate -f simulate.yaml --envtest

# Pin a specific Kubernetes version for envtest binaries
ork simulate -f simulate.yaml --envtest --k8s-version 1.31
```

## Output

```text
Simulating website/my-site

  Cycle 1:
    + deployments/my-site
    + services/my-site
    ~ status/my-site
  Cycle 2:
    ~ status/my-site
  (cycles 3–10: identical)

  ✓ Steady state at cycle 3 in 189ms
```

`+` green — resource created. `~` yellow — resource updated. `-` red — resource deleted. Consecutive identical cycles are collapsed. `status/...` appears every cycle — that is expected.

`--cycles N` runs exactly N cycles. Steady state is noted but does not stop the run.

## How it works

- Spins up a fake in-memory Kubernetes cluster using `k8s.io/client-go/kubernetes/fake`
- Runs the same `Generic Reconciler` used in production
- Records all `create`, `update`, `delete`, `patch` operations per cycle
- Advances Deployment status to `Available` after cycle 1 to unblock state machines
- Detects steady state when two consecutive cycles produce identical operations

`ork simulate` does not require a real cluster and does not read from or write to `~/.kube/config`.

---

# ork simulate init

Generate a `simulate.yaml` pre-filled with the cycle-1 create operations observed from a live reconcile run.

```bash
ork simulate init
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `katalog.yaml` | Path to the Katalog file |
| `--cr` | | `cr.yaml` | Path to the CR YAML |
| `--force` | | `false` | Overwrite an existing `simulate.yaml` |
| `--dry-run` | | `false` | Print the generated `simulate.yaml` to stdout instead of writing the file |

## Examples

```bash
# Standard layout — generates simulate.yaml in the current directory
ork simulate init

# Preview without writing
ork simulate init --dry-run

# Non-standard filenames
ork simulate init -f my-operator.yaml --cr my-cr.yaml

# Overwrite an existing simulate.yaml
ork simulate init --force
```

## Output

```text
✓ Generated simulate.yaml
  1 CRD(s), 3 op rule(s)

  Run ork simulate to verify.
```

The generated file includes `steady: true`, `noErrors: true`, and an `ops:` list of every resource created on cycle 1. Edit and refine from there — use `ork simulate --debug-ops` to see all ops across all cycles if you need to add rules beyond cycle 1.
