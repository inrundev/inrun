package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/orkspace/orkestra/pkg/utils"
)

const (
	fileKatalog    = "katalog.yaml"
	fileKomposer   = "komposer.yaml"
	fileE2e        = "e2e.yaml"
	fileSimulate   = "simulate.yaml"
	fileCrd        = "crd.yaml"
	fileCr         = "cr.yaml"
	fileReadMe     = "README.md"
	fileMakeFile   = "Makefile"
	fileDockerfile = "Dockerfile"
	fileValues     = "values.yaml"
)

// resolveKatalogPaths resolves the katalog file paths in the following order:
//
//  1. Explicit CLI paths (highest priority)
//  2. Default file paths in the working directory (katalog.yaml, komposer.yaml, etc.)
//  3. Paths defined in the Konfig (kfg.Katalog().Paths())
//
// Returns an error if no katalog file can be resolved.
func resolveKatalogPaths(cliPaths []string) ([]string, error) {
	cfgPaths := kfg.Katalog().Paths()

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
	if defaults := defaultFilePaths(); len(defaults) > 0 {
		return defaults, nil
	}

	// 3. Config-defined paths
	if len(cfgPaths) > 0 {
		return cfgPaths, nil
	}

	return nil, fmt.Errorf(errNoKatalog)
}

// defaultFilePaths returns the default katalog file if one exists in the
// current directory and no -f flag was provided. Tries katalog.yaml first,
// then komposer.yaml — the same precedence as Docker's Dockerfile / compose.yaml.
func defaultFilePaths() []string {
	for _, name := range []string{fileKatalog, fileKomposer} {
		if _, err := os.Stat(name); err == nil {
			return []string{name}
		}
	}
	return nil
}

const errNoKatalog = "no katalog.yaml or komposer.yaml found in current directory\n" +
	"pass -f <file> or create one with ork init"

// resolveKatalogFile resolves a single katalog file path from a CLI flag value.
// If flagValue is empty it falls back to defaultFilePaths(). The resolved path
// is always returned as an absolute path.
func resolveKatalogFile(flagValue string) (string, error) {
	if flagValue == "" {
		if d := defaultFilePaths(); len(d) > 0 {
			flagValue = d[0]
		}
	}
	if flagValue == "" {
		return "", fmt.Errorf(errNoKatalog)
	}
	if abs, err := filepath.Abs(flagValue); err == nil {
		return abs, nil
	}
	return flagValue, nil
}

// File helpers live in pkg/utils; these names keep existing callers unchanged.
var (
	fileExists = utils.FileExists
	isDir      = utils.IsDir
)
