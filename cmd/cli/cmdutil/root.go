package cmdutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
)

var (
	Kfg *konfig.Konfig
	Ctx context.Context
)

var RootCmd = &cobra.Command{
	Use:   "ork",
	Short: "Orkestra — Kubernetes for Everyone",
	Long: fmt.Sprintf(`
%s
Orkestra — Kubernetes for Everyone
Kompose. Konduct. OrKestrate.
`, utils.OrkestraLogoCLI),
}

func Execute(k *konfig.Konfig, c context.Context) {
	Kfg = k
	Ctx = c

	if err := RootCmd.Execute(); err != nil {
		utils.Exit(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// SilenceUsage is an option to silence usage when an error occurs.
	RootCmd.SilenceUsage = true
	// SilenceErrors is an option to quiet errors down stream.
	RootCmd.SilenceErrors = true
	RootCmd.CompletionOptions.DisableDefaultCmd = true

	// Global flags — always present in both runtime and dev builds
	RootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	RootCmd.PersistentFlags().StringSliceP("file", "f", nil, "Path(s) or URL(s) to katalog.yaml (repeatable)")
	// Dev-only flags (--kubeconfig, --verbose) and required-flag marking for
	// dev commands are registered in root_dev.go (//go:build !runtime).
}

func initConfig() {
	// Resolve log level (flag > env > default) and initialize logger
	level := resolveLogLevel(RootCmd)
	logger.Init(level)

	// Resolve kubeconfig path (flag > env > ~/.kube/config > in‑cluster)
	kubeconfig := resolveKubeconfig(RootCmd)

	// Persist resolved values into global Konfig
	if Kfg != nil {
		Kfg.Cluster().SetKubekonfigPath(kubeconfig)
	}
}

// resolveKubeconfig determines which kubeconfig to use.
// Priority: CLI flag → $KUBECONFIG → ~/.kube/config → in‑cluster.
func resolveKubeconfig(cmd *cobra.Command) string {
	if flagVal, _ := cmd.Flags().GetString("kubeconfig"); flagVal != "" {
		return flagVal
	}

	if envVal := Kfg.Cluster().KubekonfigPath(); envVal != "" {
		return envVal
	}
	if home, err := os.UserHomeDir(); err == nil {
		defaultPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}
	return ""
}

// resolveLogLevel determines the effective log level.
// Priority: --debug flag → LOG_LEVEL env → "info".
func resolveLogLevel(cmd *cobra.Command) string {
	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		return "debug"
	}

	if envVal := Kfg.Ork().LogLevel(); envVal != "" {
		return envVal
	}

	return "info"
}
