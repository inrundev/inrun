# cmd/cli

CLI command definitions for the `ork` binary. Each file registers one command or sub-command with the root Cobra command.

## Entry points

| Command | File | What it does |
|---------|------|-------------|
| `ork run` | `run.go` / `run_dev.go` | Start the runtime (reconcile loop) |
| `ork gate` | `gate.go` | Start the gateway (TLS + webhooks, cluster-only) |
| `ork generate` | `generate.go` | Generate RBAC, bundles, ConfigMaps, CRDs, docs |
| `ork validate` | `validate.go` | Validate an Orkestra Pattern |

For the full command reference see [documentation/reference/cli](../../documentation/reference/cli/index.md).

## Design docs

- [build-tags](docs/build-tags.md) — how `//go:build runtime` and `//go:build gateway` control which sub-commands ship in each image
