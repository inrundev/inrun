package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/orkspace/orkestra/pkg/utils"
)

const (
	FileKatalog    = "katalog.yaml"
	fileKomposer   = "komposer.yaml"
	FileE2e        = "e2e.yaml"
	FileSimulate   = "simulate.yaml"
	FileCrd        = "crd.yaml"
	FileCr         = "cr.yaml"
	FileReadMe     = "README.md"
	FileMakeFile   = "Makefile"
	FileDockerfile = "Dockerfile"
	FileValues     = "values.yaml"

	// A project keeps katalog.yaml at its root, what it applies to the cluster
	// (crd.yaml, cr.yaml, setup files) in manifests/, and its simulate.yaml,
	// e2e.yaml and test values in test/.
	DirManifests = "manifests"
	DirTest      = "test"
)

// FindFile returns name if it exists in the current directory, else the
// first dirs/name that exists, else "".
func FindFile(name string, dirs ...string) string {
	if FileExists(name) {
		return name
	}
	for _, d := range dirs {
		if p := filepath.Join(d, name); FileExists(p) {
			return p
		}
	}
	return ""
}

// ResolveKatalogPaths resolves the katalog file paths in the following order:
//
//  1. Explicit CLI paths (highest priority)
//  2. Default file paths in the working directory (katalog.yaml, komposer.yaml, etc.)
//  3. Paths defined in the Konfig (kfg.Katalog().Paths())
//
// Returns an error if no katalog file can be resolved.
func ResolveKatalogPaths(cliPaths []string) ([]string, error) {
	cfgPaths := Kfg.Katalog().Paths()

	// 1. CLI-provided paths — convert to absolute so all downstream relative
	// path resolutions (crdFile, crFiles, setup, Komposer imports) use the
	// file's directory as the base, not the current working directory.
	if len(cliPaths) > 0 {
		abs := make([]string, len(cliPaths))
		for i, p := range cliPaths {
			if a, err := filepath.Abs(p); err == nil {
				abs[i] = a
			} else {
				abs[i] = p
			}
		}
		return abs, nil
	}

	// 2. Default file paths
	if defaults := DefaultFilePaths(); len(defaults) > 0 {
		return defaults, nil
	}

	// 3. Config-defined paths
	if len(cfgPaths) > 0 {
		return cfgPaths, nil
	}

	return nil, fmt.Errorf(ErrNoKatalog)
}

// DefaultFilePaths returns the default katalog file if one exists in the
// current directory and no -f flag was provided. Tries katalog.yaml first,
// then komposer.yaml — the same precedence as Docker's Dockerfile / compose.yaml.
func DefaultFilePaths() []string {
	for _, name := range []string{FileKatalog, fileKomposer} {
		if _, err := os.Stat(name); err == nil {
			return []string{name}
		}
	}
	return nil
}

const ErrNoKatalog = "no katalog.yaml or komposer.yaml found in current directory\n" +
	"pass -f <file> or create one with ork init"

// ResolveKatalogFile resolves a single katalog file path from a CLI flag value.
// If flagValue is empty it falls back to defaultFilePaths(). The resolved path
// is always returned as an absolute path.
func ResolveKatalogFile(flagValue string) (string, error) {
	if flagValue == "" {
		if d := DefaultFilePaths(); len(d) > 0 {
			flagValue = d[0]
		}
	}
	if flagValue == "" {
		return "", fmt.Errorf(ErrNoKatalog)
	}
	if abs, err := filepath.Abs(flagValue); err == nil {
		return abs, nil
	}
	return flagValue, nil
}

// File helpers live in pkg/utils; these names keep existing callers unchanged.
var (
	FileExists = utils.FileExists
	IsDir      = utils.IsDir
)
