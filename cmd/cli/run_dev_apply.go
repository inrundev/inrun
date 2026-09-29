//go:build !runtime && !gateway

package cli

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	orkpkg "github.com/orkspace/orkestra/pkg/tools/cluster"
)

// applyPreRuntimeResources applies all pre-runtime resources declared in the
// merged CRD entries. This includes:
//
//  1. CRD files      (crdFile)
//  2. Waiting for CRDs to establish
//  3. CR files       (crFiles) — dev mode only
//  4. Setup files    (setup)
//
// All resources are applied in the correct order before the operator runtime
// starts. Relative paths are resolved against the katalog directory. This
// function is a no-op when running inside the cluster.
func applyPreRuntimeResources(ctx context.Context, katalogPath string, m *merger.Merger) {
	if isRunningInPod() {
		return
	}

	applyCRDFilesIfNeeded(ctx, katalogPath, m)
	waitForCRDsEstablished(ctx, m)
	applyCRFilesIfNeeded(ctx, katalogPath, m)
	applySetupIfNeeded(ctx, katalogPath, m)
}

// ensureClusterReady ensures that a Kubernetes cluster is reachable before
// starting the operator. In dev mode, this installs missing dependencies and
// creates a local Kind cluster if needed. Outside dev mode, this reports
// missing dependencies and unreachable cluster state to the user.
func ensureClusterReady(dev bool) error {
	if dev {
		if err := orkpkg.EnsureDependencies(); err != nil {
			return fmt.Errorf("installing dependencies: %w", err)
		}

		fmt.Println("\n  Cannot reach a Kubernetes cluster.")
		fmt.Printf("  Creating local Kind cluster '%s'...\n", orkpkg.KindClusterName)

		if err := orkpkg.EnsureKindCluster(orkpkg.KindClusterName, 0, ""); err != nil {
			return fmt.Errorf("setting up kind cluster: %w", err)
		}

		return nil
	}

	// non-dev mode
	if orkpkg.ClusterReachable() {
		return nil
	}

	fmt.Println("\n  Cannot reach a Kubernetes cluster.")
	fmt.Println("  Check your kubeconfig, or run with --dev to deploy to a local kind cluster.")

	var missing []string
	helm := orkpkg.HelmAvailable()
	kubectl := orkpkg.KubectlAvailable()

	if !kubectl {
		missing = append(missing, "kubectl")
	}
	if !helm {
		missing = append(missing, "helm")
	}

	if len(missing) > 0 {
		text := "these missing dependencies"
		if len(missing) == 1 {
			text = "this missing dependency"
		}

		fmt.Printf("  This will install %s:\n", text)
		for _, m := range missing {
			fmt.Printf("    • %s\n", m)
		}
		fmt.Println()
	}

	return fmt.Errorf("cluster not reachable")
}

// applyCRFilesIfNeeded applies crFiles declarations via kubectl in order before
// the runtime starts. Only runs outside the cluster (dev mode).
func applyCRFilesIfNeeded(ctx context.Context, katalogPath string, m *merger.Merger) {
	if isRunningInPod() {
		return
	}

	katalogDir := filepath.Dir(katalogPath)

	for crdName, entry := range m.All() {
		if !entry.HasCRFiles() {
			continue
		}
		for _, crFile := range entry.CRFiles {
			path := crFile
			if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
				path = filepath.Join(katalogDir, path)
			}

			out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", path).CombinedOutput()
			if err != nil {
				logger.Warn().
					Str("crd", crdName).
					Str("path", path).
					Str("output", strings.TrimSpace(string(out))).
					Err(err).
					Msg("crFile pre-apply failed (continuing)")
			} else {
				logger.Info().
					Str("crd", crdName).
					Str("path", path).
					Msg("crFile applied")
			}
		}
	}
}

// waitForCRDsEstablished waits for any CRDs that have crFiles to be Established
// in the cluster before CRs are applied. Only blocks for CRDs that need it.
func waitForCRDsEstablished(ctx context.Context, m *merger.Merger) {
	if isRunningInPod() {
		return
	}

	for crdName, entry := range m.All() {
		if len(entry.CRFiles) == 0 {
			continue
		}

		plural := entry.APITypes.Plural
		group := entry.APITypes.Group
		if plural == "" || group == "" {
			continue
		}

		crdFullName := plural + "." + group

		out, err := exec.CommandContext(ctx, "kubectl", "wait",
			"--for=condition=Established",
			"--timeout=30s",
			"crd/"+crdFullName,
		).CombinedOutput()
		if err != nil {
			logger.Warn().
				Str("crd", crdName).
				Str("name", crdFullName).
				Str("output", strings.TrimSpace(string(out))).
				Err(err).
				Msg("CRD not established in time — applying CRs anyway")
		} else {
			logger.Info().
				Str("crd", crdName).
				Str("name", crdFullName).
				Msg("CRD established")
		}
	}
}

// applyCRDFilesIfNeeded applies any crdFile declarations via kubectl before the
// operator starts. Only runs outside the cluster (dev mode). In production,
// CRDs must be pre-applied by the platform operator.
func applyCRDFilesIfNeeded(ctx context.Context, katalogPath string, m *merger.Merger) {
	if isRunningInPod() {
		return
	}

	katalogDir := filepath.Dir(katalogPath)

	for crdName, entry := range m.All() {
		if !entry.HasCRDFile() {
			continue
		}

		path := entry.CRDFile
		if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
			path = filepath.Join(katalogDir, path)
		}

		out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", path).CombinedOutput()
		if err != nil {
			logger.Warn().
				Str("crd", crdName).
				Str("path", path).
				Str("output", strings.TrimSpace(string(out))).
				Err(err).
				Msg("crdFile pre-apply failed (continuing)")
		} else {
			logger.Info().
				Str("crd", crdName).
				Str("path", path).
				Msg("crdFile applied")
		}
	}
}

// applyPatternExamples applies crd.yaml and cr.yaml from the pattern directory
// when --apply-cr is set. These are the example files shipped with the pattern —
// distinct from crdFile/crFiles declared in the katalog itself. kubectl apply
// is idempotent so overlap with applyPreRuntimeResources is safe.
func applyPatternExamples(ctx context.Context, katalogPath string, m *merger.Merger) {
	dir := filepath.Dir(katalogPath)

	crdPath := filepath.Join(dir, fileCrd)
	if fileExists(crdPath) {
		out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", crdPath).CombinedOutput()
		if err != nil {
			logger.Warn().Str("path", crdPath).Str("output", strings.TrimSpace(string(out))).Err(err).Msgf("%s apply failed", fileCrd)
		} else {
			logger.Info().Str("path", crdPath).Msgf("%s applied", fileCrd)
		}
		waitForCRDsEstablished(ctx, m)
	}

	crPath := filepath.Join(dir, fileCr)
	if fileExists(crPath) {
		out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", crPath).CombinedOutput()
		if err != nil {
			logger.Warn().Str("path", crPath).Str("output", strings.TrimSpace(string(out))).Err(err).Msgf("%s apply failed", fileCr)
		} else {
			logger.Info().Str("path", crPath).Msgf("%s applied", fileCr)
		}
	}
}

// applySetupIfNeeded applies setup YAML files via kubectl in order before
// Orkestra starts. Only runs outside the cluster (dev mode).
func applySetupIfNeeded(ctx context.Context, katalogPath string, m *merger.Merger) {
	if isRunningInPod() {
		return
	}

	katalogDir := filepath.Dir(katalogPath)

	for crdName, entry := range m.All() {
		if !entry.HasSetup() {
			continue
		}

		for _, setupFile := range entry.Setup.Apply {
			path := setupFile.Path
			if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
				path = filepath.Join(katalogDir, path)
			}

			out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", path).CombinedOutput()
			if err != nil {
				logger.Warn().
					Str("crd", crdName).
					Str("path", path).
					Str("output", strings.TrimSpace(string(out))).
					Err(err).
					Msg("setup pre-apply failed (continuing)")
			} else {
				logger.Info().
					Str("crd", crdName).
					Str("path", path).
					Msg("setup applied")
			}
		}
	}
}
