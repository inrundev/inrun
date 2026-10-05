package generate

const (
	// TypeRegistryPackage is the fixed output directory for all generated typeregistry files.
	// Both the registry and declarative hook implementations are written here.
	// The Orkestra runtime imports this package directly.
	TypeRegistryPackage = "pkg/typeregistry"

	// RegistryFile is the generated file containing:
	//   - ObjectRegistry
	//   - ListRegistry
	//   - HookRegistry
	//   - ReconcilerRegistry
	//   - RegisterScheme()
	//
	// It is regenerated on every `ork generate registry` invocation.
	RegistryFile = "zz_generated_typeregistry.go"
)
