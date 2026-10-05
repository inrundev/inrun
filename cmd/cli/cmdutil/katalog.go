//go:build !runtime && !gateway

package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/katalog/pipeline"
	"github.com/orkspace/orkestra/pkg/merger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
)

// DefaultNamespace returns the namespace to use when --namespace is not supplied.
// Reads ORK_NAMESPACE from the environment so that CLI invocations inside
// an already-configured cluster automatically target the right namespace.
func DefaultNamespace() string {
	if ns := os.Getenv("ORK_NAMESPACE"); ns != "" {
		return ns
	}
	return "orkestra-system"
}

// ParseFilePaths handles comma-separated values and returns a slice of paths
func ParseFilePaths(paths []string) []string {
	var expanded []string
	for _, p := range paths {
		// Split by comma and trim spaces
		parts := strings.Split(p, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				expanded = append(expanded, trimmed)
			}
		}
	}
	return expanded
}

type MergerOut struct {
	Merger  *merger.Merger
	CRDs    []orktypes.CRDEntry
	Katalog *katalog.Katalog
	Paths   []string
	Enabled map[string]orktypes.CRDEntry // m.Enabled()
}

func GenerateKatalog(cmd *cobra.Command) (*MergerOut, error) {
	katalogPaths, _ := cmd.Flags().GetStringSlice("file")

	expanded := ParseFilePaths(katalogPaths)
	if len(expanded) == 0 {
		expanded = DefaultFilePaths()
	}
	if len(expanded) == 0 {
		return nil, fmt.Errorf(ErrNoKatalog)
	}

	// Absolutize so merger.FirstEntryDir() is always absolute. Without this,
	// relative paths cause a double-join when populateAPITypesFromCRDFile
	// prepends katalogDir to a crdFile that was already joined once during load.
	for i, p := range expanded {
		if abs, err := filepath.Abs(p); err == nil {
			expanded[i] = abs
		}
	}

	m := merger.New(expanded...)
	if err := m.Merge(); err != nil {
		return nil, err
	}

	var kat katalog.Katalog
	kat.Spec = m.ToSpec()

	// Convert map to slice for generate functions
	crdMap := m.ToSpec().CRDs
	crds := make([]orktypes.CRDEntry, 0, len(crdMap))
	for _, c := range crdMap {
		crds = append(crds, c)
	}

	return &MergerOut{
		Merger:  m,
		CRDs:    crds,
		Katalog: &kat,
		Paths:   katalogPaths,
		Enabled: m.Enabled(),
	}, nil
}

// BuildKatalog builds the expanded Katalog using the --file flag from cmd.
func BuildKatalog(cmd *cobra.Command) (*katalog.Katalog, error) {
	m, err := GenerateKatalog(cmd)
	if err != nil {
		return nil, fmt.Errorf("generating Katalog: %w", err)
	}
	return pipeline.BuildExpanded(Kfg, m.Merger)
}
