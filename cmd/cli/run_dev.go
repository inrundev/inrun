//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/cmd/internal"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/orkspace/orkestra/pkg/tools/devserver"
	"github.com/spf13/cobra"
)

// runCmd is the full dev build version of ork run.
// The production version lives in run.go (//go:build runtime).
// This file owns the command registration for dev builds and layers
// the --dev cluster-setup behaviour on top of the core production logic.
var runCmd = &cobra.Command{
	Use:   "run [<name>:<version>]",
	Short: "Start the Orkestra Runtime",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, _ := cmd.Flags().GetBool("dev")
		devServer, _ := cmd.Flags().GetBool("dev-server")
		devServerPort, _ := cmd.Flags().GetInt("dev-server-port")
		useKomposer, _ := cmd.Flags().GetBool("use-komposer")
		refresh, _ := cmd.Flags().GetBool("refresh")
		applyCR, _ := cmd.Flags().GetBool("apply-cr")

		// Handle dev mode cluster creation
		if err := ensureClusterReady(dev); err != nil {
			return err
		}

		// Start the mock dev server before the runtime if requested.
		if devServer {
			if err := devserver.Start(devServerPort); err != nil {
				return fmt.Errorf("starting dev server: %w", err)
			}
		}

		// If a positional OCI ref is given, pull it and resolve to a local path.
		paths, _ := cmd.Flags().GetStringSlice("file")
		if len(args) == 1 && registry.IsOCIRef(args[0]) {
			p, err := resolveOCIRunPath(cmd.Context(), args[0], useKomposer, refresh)
			if err != nil {
				return err
			}
			paths = append([]string{p}, paths...)
		}

		// Resolve katalog paths
		paths, err := resolveKatalogPaths(paths)
		if err != nil {
			return err
		}

		// Merge katalogs
		m := merger.New(paths...)
		if err := m.Merge(); err != nil {
			return fmt.Errorf("merging katalogs: %w", err)
		}

		logger.Debug().
			Strs("katalogs", paths).
			Int("total", m.Count()).
			Int("enabled", m.EnabledCount()).
			Msg("katalogs merged")

		// Apply declared crdFile, crFiles and setup paths before handing off to the runtime.
		if len(paths) > 0 {
			applyPreRuntimeResources(cmd.Context(), paths[0], m)
			if applyCR {
				applyPatternExamples(cmd.Context(), paths[0], m)
			}
		}

		// Run the runtime
		internal.KonductRuntime(kfg, m, ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().Bool("dev", false, "Create a local Kind cluster if none is reachable (development only)")
	runCmd.Flags().Bool("dev-server", false, "Start the mock dev server for external: examples (no real services needed)")
	runCmd.Flags().Int("dev-server-port", devserver.Port, "Port for the mock dev server")
	runCmd.Flags().Bool("use-komposer", false, "Use komposer.yaml from the pulled pattern instead of katalog.yaml")
	runCmd.Flags().Bool("refresh", false, "Re-pull the pattern from the registry even if already cached")
	runCmd.Flags().Bool("apply-cr", false, "Apply crd.yaml and cr.yaml from the remote pattern directory before starting the runtime")
}
