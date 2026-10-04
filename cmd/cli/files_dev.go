//go:build !runtime && !gateway

package cli

// Add any function that requires dev-only tools or symbols (startSpinner, successMark,
// pkg/registry, etc.) here. This file is excluded from runtime and gateway builds.

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/orkspace/orkestra/pkg/registry"
)

// resolveOCIRunPath resolves an OCI pattern reference to a local katalog or
// komposer path, pulling to the cache if necessary.
func resolveOCIRunPath(ctx context.Context, ref string, useKomposer, refresh bool) (string, error) {
	r, err := registry.ResolveForKind(ref, registry.KatalogKind)
	if err != nil {
		return "", fmt.Errorf("invalid reference: %w", err)
	}

	if !r.IsCached() || refresh {
		client, err := registry.NewClient()
		if err != nil {
			return "", fmt.Errorf("initializing registry client: %w", err)
		}
		fmt.Printf("Pulling %s\n  → %s\n", r.ShortName(), r.String())
		spin := startSpinner("Downloading...")
		if _, err := client.Pull(ctx, r, refresh); err != nil {
			spin.Failure()
			return "", fmt.Errorf("pull failed: %w", err)
		}
		spin.Stop()
		fmt.Printf("  %s Cached\n", successMark())
	}

	cacheDir, err := r.CachePath()
	if err != nil {
		return "", err
	}

	target := fileKatalog
	if useKomposer {
		target = fileKomposer
	}
	p := filepath.Join(cacheDir, target)
	if !fileExists(p) {
		return "", fmt.Errorf("%s not found in cached pattern %s", target, r.ShortName())
	}
	return p, nil
}
