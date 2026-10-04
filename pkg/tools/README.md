# pkg/tools

CLI-support packages. None of these run in the cluster.

| Sub-package | CLI command | What it does |
|-------------|-------------|--------------|
| [cluster/](cluster/README.md)   | (shared)      | Cluster-facing utilities: Helm setup, kind, GVK helpers, health checks |
| [generate/](generate/README.md) | `ork generate` | Bundle, RBAC, and type registry generation |
| [migrate/](migrate/README.md)   | `ork migrate`  | Katalog file migration between schema versions |
| [devserver/](devserver/README.md) | `ork dev`    | Local Control Center dev server for template development |
| [proxy/](proxy/README.md)       | `ork proxy`    | Port-forward discovery for local development |
