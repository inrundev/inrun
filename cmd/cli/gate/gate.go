// gateway.go — Production gateway entrypoint.
//
// This command has one responsibility: run Orkestra Gateway.
// It performs exactly three tasks:
//  1. Load all provided katalog files.
//  2. Merge them into a single resolved Katalog.
//  3. Start the Orkestra gateway using the merged result.
//
// All development‑only behavior are excluded from production
// builds via build tags.

//go:build gateway

package gate

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/cmd/cli/cmdutil"

	"github.com/orkspace/orkestra/cmd/internal"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/spf13/cobra"
)

var gatewayCmd = &cobra.Command{
	Use:   "gate",
	Short: "Start the Orkestra gateway (TLS + admission webhooks, cluster-only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := cmd.Flags().GetStringSlice("file")
		if len(paths) == 0 {
			paths = cmdutil.DefaultFilePaths()
		}
		if len(paths) == 0 {
			paths = cmdutil.Kfg.Katalog().Paths()
		}
		if len(paths) == 0 {
			return fmt.Errorf(cmdutil.ErrNoKatalog)
		}

		m := merger.New(paths...)
		if err := m.Merge(); err != nil {
			return fmt.Errorf("merging katalogs: %w", err)
		}

		logger.Debug().
			Str("katalogs", strings.Join(paths, ", ")).
			Int("total", m.Count()).
			Int("enabled", m.EnabledCount()).
			Msg("katalogs merged")

		internal.KonductGateway(cmdutil.Kfg, m, cmdutil.Ctx)
		return nil
	},
}

func init() {
	cmdutil.RootCmd.AddCommand(gatewayCmd)
	gatewayCmd.Flags().StringSliceP("file", "f", nil, "Path(s) or URL(s) to crd-katalog.yaml (repeatable)")
}
