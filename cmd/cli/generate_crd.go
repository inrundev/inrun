// cmd/ClientConfig/generate_crd.go
//
// ork generate crd  — derives a CRD from a Katalog
// ork generate cr   — derives an example CR from a Katalog
//
// The Katalog is the single source of truth. The CRD and CR are generated
// from what is already declared:
//
//	apiTypes    → group, version, kind, plural, scope, printer columns
//	validation  → required spec fields (deny + exists rules)
//	mutation    → optional fields with defaults, type inference
//	templates   → additional spec fields referenced as {{ .spec.* }}
//	status      → status subresource schema, printer columns
//	conversion  → webhook config (when conversion paths are declared)
//
// Usage:
//
//	ork generate crd --file katalog.yaml -o crd.yaml
//	ork generate cr  --file katalog.yaml -o cr.yaml
//
// Multiple CRDs in one Katalog:
//
//	ork generate crd --file katalog.yaml --crd pipeline -o pipeline-crd.yaml
//	ork generate cr  --file katalog.yaml --crd pipeline -o pipeline-cr.yaml

//go:build !runtime && !gateway

package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/tools/generate"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// generateCRDCmd implements: ork generate crd
var generateCRDCmd = &cobra.Command{
	Use:   "crd",
	Short: "Generate a CRD from a Katalog",
	Long: `Derive a CustomResourceDefinition YAML from a Katalog file.

The Katalog is the single source of truth. The generated CRD includes:
  - OpenAPI schema derived from validation rules and template expressions
  - Required fields from deny + exists validation rules
  - Optional fields with defaults from mutation rules
  - Status subresource schema from status.fields declarations
  - AdditionalPrinterColumns from status fields (phase first)
  - Conversion webhook config when conversion paths are declared

Examples:
  # Single CRD Katalog
  ork generate crd --file katalog.yaml -o crd.yaml

  # Multi-CRD Katalog — generate for one specific CRD
  ork generate crd --file katalog.yaml --crd pipeline -o pipeline-crd.yaml

  # Multi-CRD — generate all (outputs pipeline-crd.yaml, website-crd.yaml, ...)
  ork generate crd --file katalog.yaml --all -o ./crds/`,
	RunE: runGenerateCRD,
}

// generateCRCmd implements: ork generate cr
var generateCRCmd = &cobra.Command{
	Use:   "cr",
	Short: "Generate an example CR from a Katalog",
	Long: `Derive an example CustomResource YAML from a Katalog file.

The generated CR includes:
  - Required fields filled with typed placeholders
  - Optional fields showing their default values
  - Comments explaining each field (--annotate flag)

Examples:
  ork generate cr --file katalog.yaml -o cr.yaml
  ork generate cr --file katalog.yaml --crd pipeline -o pipeline-cr.yaml`,
	RunE: runGenerateCR,
}

func init() {
	// crd flags
	generateCRDCmd.Flags().StringSliceP("file", "f", nil, "Path(s) to Katalog YAML file(s)")
	generateCRDCmd.Flags().StringP("output", "o", "", "Output file or directory (default: stdout)")
	generateCRDCmd.Flags().String("crd", "", "CRD name to generate (default: first CRD)")
	generateCRDCmd.Flags().Bool("all", false, "Generate CRDs for all CRDs in the Katalog")
	generateCRDCmd.MarkFlagRequired("katalog")

	// cr flags
	generateCRCmd.Flags().StringSliceP("file", "f", nil, "Path(s) to Katalog YAML file(s)")
	generateCRCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	generateCRCmd.Flags().String("crd", "", "CRD name to generate CR for (default: first CRD)")
	generateCRCmd.MarkFlagRequired("katalog")

	generateCmd.AddCommand(generateCRDCmd)
	generateCmd.AddCommand(generateCRCmd)
}

func runGenerateCRD(cmd *cobra.Command, _ []string) error {
	output, _ := cmd.Flags().GetString("output")
	crdName, _ := cmd.Flags().GetString("crd")
	all, _ := cmd.Flags().GetBool("all")

	gen, err := generateKatalog(cmd)
	if err != nil {
		return err
	}

	crds := filterByName(gen.m.Enabled(), crdName, all)
	if len(crds) == 0 {
		if crdName != "" {
			return fmt.Errorf("CRD %q not found in Katalog", crdName)
		}
		return fmt.Errorf("no enabled CRDs found in Katalog")
	}

	for _, crd := range crds {
		gen := generate.NewCRDGenerator(crd)
		obj, err := gen.Generate()
		if err != nil {
			return fmt.Errorf("generating CRD for %q: %w", crd.Name, err)
		}

		data, err := yaml.Marshal(obj)
		if err != nil {
			return err
		}

		if output == "" {
			fmt.Fprint(cmd.OutOrStdout(), "---\n")
			cmd.OutOrStdout().Write(data)
			continue
		}

		// Directory output — one file per CRD
		outPath := output
		if all || isDir(output) {
			if err := os.MkdirAll(output, 0755); err != nil {
				return fmt.Errorf("creating output directory %s: %w", output, err)
			}
			outPath = filepath.Join(output, strings.ToLower(crd.Name)+"-crd.yaml")
		}

		if err := os.WriteFile(outPath, append([]byte("---\n"), data...), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Generated: %s\n", outPath)
	}

	return nil
}

func runGenerateCR(cmd *cobra.Command, _ []string) error {
	katalogPaths, _ := cmd.Flags().GetStringSlice("file")
	output, _ := cmd.Flags().GetString("output")
	crdName, _ := cmd.Flags().GetString("crd")

	gen, err := generateKatalog(cmd)
	if err != nil {
		return err
	}

	crds := filterByName(gen.m.Enabled(), crdName, false)
	if len(crds) == 0 {
		if crdName != "" {
			return fmt.Errorf("CRD %q not found in Katalog", crdName)
		}
		return fmt.Errorf("no enabled CRDs found in Katalog")
	}

	var out [][]byte
	for _, crd := range crds {
		gen := generate.NewCRGenerator(crd)
		cr := gen.Generate()

		data, err := yaml.Marshal(cr)
		if err != nil {
			return err
		}

		header := fmt.Sprintf("# Example CR for %s\n# Generated from: %s\n# Apply: kubectl apply -f cr.yaml\n---\n",
			crd.APITypes.Kind, katalogPaths[0])
		out = append(out, append([]byte(header), data...))
	}

	combined := bytes.Join(out, []byte("\n"))

	if output == "" {
		cmd.OutOrStdout().Write(combined)
		return nil
	}

	if err := os.WriteFile(output, combined, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", output, err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Generated: %s\n", output)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// filterByName returns CRD entries from the map filtered by name.
// If name is non-empty, only the matching entry is returned (error if missing).
// If all is false and name is empty, only the first entry (alphabetically) is returned.
// If all is true and name is empty, all entries are returned.
func filterByName(crds map[string]orktypes.CRDEntry, name string, all bool) []orktypes.CRDEntry {
	if name != "" {
		name = strings.ToLower(name)
		if c, ok := crds[name]; ok {
			return []orktypes.CRDEntry{c}
		}
		return nil
	}
	result := make([]orktypes.CRDEntry, 0, len(crds))
	for _, c := range crds {
		result = append(result, c)
	}
	if !all && len(result) > 0 {
		return result[:1]
	}
	return result
}
