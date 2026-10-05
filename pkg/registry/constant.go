package registry

const (
	KatalogKind PatternKind = "Katalog" // Katalog-based operator pattern
	MotifKind   PatternKind = "Motif"   // Reusable resource primitive
	UnknownKind PatternKind = ""

	// MediaType is the OCI pattern media type for Orkestra patterns.
	MediaType = "application/vnd.orkestra.pattern.v1+tar+gzip"

	// indexMediaType is the OCI media type for the shared registry index.
	indexMediaType = "application/vnd.orkestra.index.v1+json"

	// FileKatalog is the required operator declaration file.
	FileKatalog = "katalog.yaml"

	// FileKomposer is the optional multi-operator composition file.
	FileKomposer = "komposer.yaml"

	// FileMotif is the required motif declaration file.
	FileMotif = "motif.yaml"

	// FileCRD is the required CRD schema file.
	FileCRD = "crd.yaml"

	// FileReadme is the human documentation file.
	FileReadme = "README.md"

	// FileCR is the example CR file.
	FileCR = "cr.yaml"

	// FileE2E is the E2E test definition file for a Katalog pattern.
	FileE2E = "e2e.yaml"

	// FileSimulate is the simulate spec file for a Katalog pattern.
	FileSimulate = "simulate.yaml"

	// FileIntentYAML and FileIntentJSON are the serve play intent files for a Katalog pattern.
	FileIntentYAML = "intent.yaml"
	FileIntentJSON = "intent.json"

	// DirManifests and DirTest group a pattern's files: crd.yaml and cr.yaml
	// in manifests/, simulate.yaml and e2e.yaml in test/. The root is also
	// accepted for each.
	DirManifests = "manifests"
	DirTest      = "test"

	// FileGoMod, FileGoSum, and FileMakefile are the typed operator build files.
	// Present only in typed (hooks/constructor) patterns.
	FileGoMod    = "go.mod"
	FileGoSum    = "go.sum"
	FileMakefile = "Makefile"

	// DefaultKatalogRegistry is the official OCI path for Katalog patterns.
	DefaultKatalogRegistry = "ghcr.io/orkspace/orkestra-registry/patterns/katalogs"

	// DefaultMotifRegistry is the official OCI path for Motif patterns.
	DefaultMotifRegistry = "ghcr.io/orkspace/orkestra-registry/patterns/motifs"

	// DefaultPatternRegistry is an alias for DefaultKatalogRegistry.
	DefaultPatternRegistry = DefaultKatalogRegistry

	// EnvPatternRegistry overrides the default katalog registry path.
	EnvPatternRegistry = "ORK_REGISTRY"

	// EnvMotifRegistry overrides the default motif registry path.
	EnvMotifRegistry = "ORK_MOTIFS_REGISTRY"

	// EnvRegistry is an alias for EnvPatternRegistry.
	EnvRegistry = EnvPatternRegistry

	// CacheDir is the local cache directory for pulled artifacts.
	// Resolved relative to the user's home directory.
	CacheDir = ".orkestra/registry"

	// HelmGitCacheDir is the local cache directory for git-sourced Helm charts.
	HelmGitCacheDir = ".orkestra/helm/git"

	// HelmRepoCacheDir is the local cache directory for remote Helm repository charts.
	HelmRepoCacheDir = ".orkestra/helm/repo"

	// FileCacheDir is the local cache directory for remote file fetches (https://).
	FileCacheDir = ".orkestra/files"
)
