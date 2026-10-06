// Package generate turns a Catalog into files a project or cluster needs.
//
// TypeRegistry writes the Go type registry that typed operators compile
// against. RBAC, ConfigMap and RenderBundle produce the cluster install
// manifests (permissions, the embedded Catalog, or both in one file).
// CatalogScaffold and the Write* helpers scaffold a new pattern:
// catalog.yaml, Makefile, Dockerfile, README, simulate.yaml and e2e.yaml.
package generate
