//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/orkspace/orkestra/pkg/version"
	"github.com/spf13/cobra"
)

var (
	ccPort          string
	ccURLs          string
	ccRefresh       string
	ccLogLevel      string
	ccIgnoreDefault bool
)

func init() {
	controlCmd.Flags().StringVarP(&ccPort, "port", "p", "8081", "Port to run the Control Center on")
	controlCmd.Flags().BoolVarP(&ccIgnoreDefault, "ignore-default", "i", false, "Do not add the default localhost:8080 URL; start with no instances")
	controlCmd.Flags().StringVarP(&ccURLs, "urls", "u", "http://localhost:8080", "Comma-separated list of Orkestra runtime URLs")
	controlCmd.Flags().StringVar(&ccRefresh, "refresh", "10s", "Refresh interval for fetching Katalogs")
	controlCmd.Flags().StringVar(&ccLogLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	controlCmd.AddCommand(controlVersionCmd)
	rootCmd.AddCommand(controlCmd)

	// Shadow global flags so they don't appear under `ork control`
	controlCmd.Flags().StringSlice("katalog", nil, "")
	controlCmd.Flags().MarkHidden("katalog")
	shadowGlobalCommandFlags(controlCmd)
}

var controlCmd = &cobra.Command{
	Use:   "control",
	Short: "Start the Orkestra Control Center",
	Long: `Start the Orkestra Control Center web UI.

The Control Center provides a web-based interface for monitoring
multiple Orkestra runtime and gateway instances.

Examples:
  # Start with default settings (port 8081, localhost:8080)
  ork control

  # Start on custom port with multiple instances
  ork control --port 9090 --urls "http://localhost:8080,http://localhost:8082"

  # Start with debug logging
  ork control --log-level debug --refresh 5s

  # Monitor remote instances
  ork control --urls "https://orkestra.prod.internal:8080,https://orkestra.staging:8080"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return startControlCenter()
	},
}

var controlVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show Control Center version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showControlCenterVersion()
	},
}

func startControlCenter() error {
	ccPath, err := findControlCenterBinary()
	if err != nil {
		fmt.Fprintln(os.Stderr, "[ork] orkcc not found, attempting installation...")
		if err := installControlCenterBinary(); err != nil {
			return fmt.Errorf("failed to install orkcc: %w", err)
		}
		ccPath, err = findControlCenterBinary()
		if err != nil {
			return fmt.Errorf("orkcc still not found after installation")
		}
	}

	fmt.Printf("[ork] Starting Control Center: %s -p %s -u %s\n", ccPath, ccPort, ccURLs)

	// Direct flags, no "start" subcommand
	args := []string{
		"-p", ccPort,
		"-u", ccURLs,
		"--refresh", ccRefresh,
		"--log-level", ccLogLevel,
	}

	cmd := exec.Command(ccPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Run the command and let it take over the terminal
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("control center exited with error: %w", err)
	}

	return nil
}

func showControlCenterVersion() error {
	ccPath, err := findControlCenterBinary()
	if err != nil {
		return fmt.Errorf("orkcc not found: %w", err)
	}

	cmd := exec.Command(ccPath, "--version")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func findControlCenterBinary() (string, error) {
	// Look in PATH first
	if path, err := exec.LookPath("orkcc"); err == nil {
		return path, nil
	}

	// Look next to the ork binary
	orkPath, err := exec.LookPath("ork")
	if err == nil {
		dir := strings.TrimSuffix(orkPath, "/ork")
		candidate := dir + "/orkcc"
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Look in ~/.orkestra/bin
	home, err := os.UserHomeDir()
	if err == nil {
		candidate := home + "/.orkestra/bin/orkcc"
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("orkcc not found in PATH, ~/.orkestra/bin, or next to ork binary")
}

func installControlCenterBinary() error {
	ver := version.Short()
	if !strings.HasPrefix(ver, "v") {
		return fmt.Errorf("orkcc not found — build it manually: make orkcc")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	binDir := home + "/.orkestra/bin"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	goos := runtime.GOOS
	platform := goos + "_" + runtime.GOARCH
	archive := fmt.Sprintf("orkcc_%s.tar.gz", platform)
	url := fmt.Sprintf("https://github.com/orkspace/orkestra/releases/download/%s/%s", ver, archive)
	dest := binDir + "/orkcc"
	if goos == "windows" {
		dest += ".exe"
	}

	fmt.Printf("[ork] Downloading %s...\n", archive)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("orkcc %s not found in release %s (HTTP %d) — build manually: make orkcc", platform, ver, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(binDir, ".orkcc.tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := extractBinaryFromTarGz(resp.Body, "orkcc", tmp); err != nil {
		tmp.Close()
		return fmt.Errorf("extract failed: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return err
	}
	fmt.Printf("[ork] orkcc installed to %s\n", dest)
	return nil
}
