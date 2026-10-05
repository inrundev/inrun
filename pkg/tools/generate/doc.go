// Package generate turns a Katalog into files a project or cluster needs.
//
// TypeRegistry writes the Go type registry that typed operators compile
// against. RBAC, ConfigMap and RenderBundle produce the cluster install
// manifests (permissions, the embedded Katalog, or both in one file).
// KatalogScaffold and the Write* helpers scaffold a new pattern:
// katalog.yaml, Makefile, Dockerfile, README, simulate.yaml and e2e.yaml.
package generate
