# Runners — Developer Documentation

`pkg/runtime/runners/` contains one file per Kubernetes resource type. Each runner applies a resolved, already-expanded list of template sources to the cluster.

## Documents

| File | What it covers |
|------|----------------|
| [01-runner-contract.md](01-runner-contract.md) | The canonical runner shape: activeNames pre-pass, main loop sections A–B, signature variations, error format |
| [02-garbage-collection.md](02-garbage-collection.md) | How Orkestra owner references drive Kubernetes garbage collection |

For issues and known limitations, see [issues.md](issues.md).
