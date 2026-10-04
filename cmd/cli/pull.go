//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/orkspace/orkestra/pkg/registry/motif"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

// ── pull ──────────────────────────────────────────────────────────────────────

var pullCmd = &cobra.Command{
	Use:   "pull [<name>:<version>]",
	Short: "Pull a pattern to the local cache",
	Args:  cobra.RangeArgs(0, 1),
	Example: `  ork pull postgres:v14
  ork pull oci://ghcr.io/myorg/patterns/redis:v7
  ork pull -f katalog.yaml
  ork pull -f komposer.yaml
  ork pull postgres:v14 --refresh`,
	RunE: func(cmd *cobra.Command, args []string) error {
		refresh, _ := cmd.Flags().GetBool("refresh")
		outDir, _ := cmd.Flags().GetString("out")
		filePath, _ := cmd.Flags().GetString("file")

		if filePath != "" {
			return pullFromFile(cmd, filePath, refresh)
		}

		if len(args) == 0 {
			return fmt.Errorf("provide a reference (e.g. postgres:v14) or --file <katalog.yaml>")
		}

		isMotif, _ := cmd.Flags().GetBool("motif")
		kind := registry.KatalogKind
		if isMotif {
			kind = registry.MotifKind
		}
		ref, err := registry.ResolveForKind(args[0], kind)
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		if ref.IsCached() && !refresh {
			cacheDir, _ := ref.CachePath()
			fmt.Printf("  %s Already cached\n", successMark())
			fmt.Printf("  → %s\n", cacheDir)
			printPullSuggestions(ref, cacheDir)
			return nil
		}

		fmt.Printf("Pulling %s\n  → %s\n", ref.ShortName(), ref.String())
		spin := startSpinner("Downloading...")
		cacheDir, err := client.Pull(cmd.Context(), ref, refresh)
		if err != nil {
			spin.Failure()
			return fmt.Errorf("pull failed: %w", err)
		}
		spin.Stop()

		if outDir != "" {
			if err := copyDir(cacheDir, outDir); err != nil {
				return fmt.Errorf("extracting to %s: %w", outDir, err)
			}
			fmt.Printf("  %s Extracted to %s\n", successMark(), outDir)
			if spec, err := registry.SpecFor(kind); err == nil {
				if meta, err := registry.LoadPatternMeta(cacheDir, spec); err == nil && meta.Deprecated != nil {
					printPatternDeprecation(meta.Deprecated)
				}
			}
			return nil
		}

		fmt.Printf("  %s Cached at %s\n", successMark(), cacheDir)
		printPullSuggestions(ref, cacheDir)

		if !isMotif {
			pullMotifDeps(cacheDir)
			notifyTypedPull(cacheDir)
		}

		if spec, err := registry.SpecFor(kind); err == nil {
			if meta, err := registry.LoadPatternMeta(cacheDir, spec); err == nil && meta.Deprecated != nil {
				printPatternDeprecation(meta.Deprecated)
			}
		}

		return nil
	},
}

func init() {
	pullCmd.Flags().Bool("refresh", false, "Bypass local cache and re-pull from registry")
	pullCmd.Flags().StringP("out", "o", "", "Extract pulled pattern to this directory")
	pullCmd.Flags().StringP("file", "f", "", "Pull all OCI imports from a katalog or komposer file")
	pullCmd.Flags().BoolP("motif", "m", false, "Resolve as a motif (uses ORK_MOTIFS_REGISTRY)")
	rootCmd.AddCommand(pullCmd)

	// Shadow global flags so they don't appear under `ork pull`
	shadowGlobalCommandFlags(pullCmd)
}

// notifyTypedPull prints a build note when the pulled artifact is a typed operator.
func notifyTypedPull(cacheDir string) {
	if _, err := os.Stat(joinPath(cacheDir, registry.FileGoMod)); err != nil {
		return
	}
	_, hasMakefile := os.Stat(joinPath(cacheDir, registry.FileMakefile))
	fmt.Printf("  ↳ Typed operator — requires a custom runtime\n")
	printTypedBuildSteps(hasMakefile == nil)
}

// pullMotifDeps reads the katalog.yaml in cacheDir and pulls any OCI motif
// imports it declares. Warnings are printed but do not fail the main pull.
func pullMotifDeps(katalogCacheDir string) {
	katalogFile := joinPath(katalogCacheDir, registry.FileKatalog)
	if _, err := os.Stat(katalogFile); err != nil {
		return
	}

	imports, err := registry.ExtractOCIImports(katalogFile)
	if err != nil || len(imports.MotifImports) == 0 {
		return
	}

	if imports.Empty() {
		return
	}

	fmt.Printf("\nPulling motif dependencies...\n")
	for _, imp := range imports.MotifImports {
		spin := startSpinner(imp.Motif)
		if motif.PullImport(&imp) == nil {
			spin.Stop()
			fmt.Printf("  %s %s\n", successMark(), imp.Motif)
		} else {
			spin.Failure()
			fmt.Printf("  %s %s (pull failed — will retry on next use)\n", warningMark(), imp.Motif)
		}
	}
}

// pullFromFile extracts all OCI refs from a katalog or komposer file and pulls
// each one. Motif imports (Katalog) and registry imports (Komposer) are both
// handled, including bare-name shorthands without an oci:// prefix.
func pullFromFile(cmd *cobra.Command, filePath string, refresh bool) error {
	imports, err := registry.ExtractOCIImports(filePath)
	if err != nil {
		return fmt.Errorf("reading imports from %s: %w", filePath, err)
	}
	if imports.Empty() {
		fmt.Printf("  %s No OCI imports found in %s\n", successMark(), filePath)
		return nil
	}

	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("initializing client: %w", err)
	}

	var errs []string

	for _, imp := range imports.MotifImports {
		fmt.Printf("Pulling motif %s...\n", imp.Motif)
		if err := motif.PullImport(&imp); err != nil {
			fmt.Printf("  %s %v\n", failureMark(), err)
			errs = append(errs, err.Error())
		} else {
			fmt.Printf("  %s %s\n", successMark(), imp.Motif)
		}
	}

	for _, src := range imports.RegistrySources {
		cleanURL, version := src.ResolvedURL()
		cleanURL = strings.TrimPrefix(cleanURL, "oci://")
		ref, err := registry.Resolve(cleanURL + ":" + version)
		if err != nil {
			fmt.Printf("  %s resolving %s: %v\n", failureMark(), src.URL, err)
			errs = append(errs, err.Error())
			continue
		}
		if ref.IsCached() && !refresh {
			cacheDir, _ := ref.CachePath()
			fmt.Printf("  %s Already cached: %s\n", successMark(), ref.ShortName())
			pullMotifDeps(cacheDir)
			continue
		}
		fmt.Printf("Pulling %s\n  → %s\n", ref.ShortName(), ref.String())
		spinRef := startSpinner("Downloading...")
		cacheDir, err := client.Pull(cmd.Context(), ref, refresh)
		if err != nil {
			spinRef.Failure()
			errs = append(errs, err.Error())
		} else {
			spinRef.Stop()
			fmt.Printf("  %s %s\n", successMark(), ref.ShortName())
			pullMotifDeps(cacheDir)
		}
	}

	// ── Helm sources ─────────────────────────────────────────────────────────
	helmImports, err := registry.ExtractHelmAndFileImports(filePath)
	if err == nil && !helmImports.Empty() {
		for _, src := range helmImports.HelmSources {
			label := src.Repo + "/" + src.Chart + "@" + src.Version
			spin := startSpinner(label)
			if cacheErr := merger.WarmHelmSource(src, refresh); cacheErr != nil {
				spin.Failure()
				errs = append(errs, fmt.Sprintf("helm %s: %v", label, cacheErr))
			} else {
				spin.Stop()
				fmt.Printf("  %s %s\n", successMark(), label)
			}
		}

		// ── Remote file sources ───────────────────────────────────────────────
		for _, url := range helmImports.RemoteFiles {
			spin := startSpinner(url)
			if refresh {
				utils.InvalidateFileCache(url)
			}
			if _, fetchErr := utils.LoadFileWithAuthRefresh(url, nil, refresh); fetchErr != nil {
				spin.Failure()
				errs = append(errs, fmt.Sprintf("file %s: %v", url, fetchErr))
			} else {
				spin.Stop()
				fmt.Printf("  %s %s\n", successMark(), url)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d pull(s) failed:\n  %s", len(errs), strings.Join(errs, "\n  "))
	}
	return nil
}
