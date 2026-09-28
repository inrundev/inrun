# reconcilers

Reconciler implementations for the Orkestra kordinator.

Each sub-package is a self-contained reconciler type that satisfies
`domain.Reconciler`. The kordinator wires the right one at startup
based on the katalog's `operatorBox.reconcile` configuration.

## Packages

- **generic** — the declarative reconciler; operators express intent through the katalog and Orkestra manages the full reconciliation loop.

- **remote** — transport-based reconcilers that delegate to an external
  service instead of local Go code.
  - **http** — HTTP/JSON transport; any HTTP server can act as a reconciler.
  - **grpc** — gRPC transport _(planned)_.
