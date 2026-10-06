//go:build !runtime && !gateway

package cmdutil

const (
	SchemaRefCatalog       = "https://inrun.dev/docs/reference/schema/catalog/"
	SchemaRefStack         = "https://inrun.dev/docs/reference/schema/stack/"
	SchemaRefModule        = "https://inrun.dev/docs/reference/schema/module/"
	SchemaRefE2E           = "https://inrun.dev/docs/reference/schema/e2e/"
	SchemaRefE2EImports    = "https://inrun.dev/docs/reference/schema/e2e/imports/"
	SchemaRefSimulate      = "https://inrun.dev/docs/reference/schema/simulate/"
	SchemaRefSimulateSuite = "https://inrun.dev/docs/reference/schema/simulate/#aggregator-form"
)
