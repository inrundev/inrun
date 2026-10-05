//go:build !runtime && !gateway

package registry

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/orkspace/orkestra/cmd/cli/cmdutil"

	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/spf13/cobra"
)

// ── patterns ──────────────────────────────────────────────────────────────────

var patternsCmd = &cobra.Command{
	Use:   "patterns [registry-url]",
	Short: "List available patterns in the registry",
	Args:  cobra.MaximumNArgs(1),
	Example: `  ork patterns
  ork patterns --motifs
  ork patterns --katalogs
  ork patterns oci://ghcr.io/mycompany/patterns
  ork patterns --tag database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		onlyKatalogs, _ := cmd.Flags().GetBool("katalogs")
		onlyMotifs, _ := cmd.Flags().GetBool("motifs")

		client, err := registry.NewClient()
		if err != nil {
			return fmt.Errorf("initializing client: %w", err)
		}

		var entries []registry.PatternEntry
		var latestUpdatedAt string

		if len(args) > 0 {
			idx, err := client.List(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("listing patterns: %w", err)
			}
			entries = idx.Entries
			latestUpdatedAt = idx.UpdatedAt
		} else {
			var listErrs []string
			if !onlyMotifs {
				patURL := os.Getenv(registry.EnvPatternRegistry)
				if patURL == "" {
					patURL = registry.DefaultPatternRegistry
				}
				idx, err := client.List(cmd.Context(), patURL)
				if err != nil {
					listErrs = append(listErrs, fmt.Sprintf("  patterns: %s", registryErrSummary(err)))
				} else if idx != nil {
					entries = append(entries, idx.Entries...)
					if idx.UpdatedAt > latestUpdatedAt {
						latestUpdatedAt = idx.UpdatedAt
					}
				}
			}
			if !onlyKatalogs {
				motifURL := os.Getenv(registry.EnvMotifRegistry)
				if motifURL == "" {
					motifURL = registry.DefaultMotifRegistry
				}
				idx, err := client.List(cmd.Context(), motifURL)
				if err != nil {
					listErrs = append(listErrs, fmt.Sprintf("  motifs:   %s", registryErrSummary(err)))
				} else if idx != nil {
					entries = append(entries, idx.Entries...)
					if idx.UpdatedAt > latestUpdatedAt {
						latestUpdatedAt = idx.UpdatedAt
					}
				}
			}
			if len(listErrs) > 0 {
				fmt.Fprintf(os.Stderr, "warning: registry listing failed:\n")
				for _, e := range listErrs {
					fmt.Fprintln(os.Stderr, e)
				}
				fmt.Fprintf(os.Stderr, "hint: try logging in with: docker login ghcr.io\n\n")
				if len(entries) == 0 {
					return nil
				}
			}
		}

		label := "Orkestra Registry"
		switch {
		case onlyKatalogs:
			label = "Orkestra Katalogs"
		case onlyMotifs:
			label = "Orkestra Motifs"
		}
		fmt.Printf("\n%s\n", label)
		fmt.Printf("%s\n", strings.Repeat("─", 57))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "\tNAME\tLATEST\tKIND\tE2E\tTAGS\tDESCRIPTION")

		count := 0
		for _, e := range entries {
			if tag != "" && !cmdutil.ContainsTag(e.Tags, tag) {
				continue
			}
			k := e.Kind
			if onlyKatalogs && k != registry.KatalogKind.ToString() {
				continue
			}
			if onlyMotifs && k != registry.MotifKind.ToString() {
				continue
			}
			tags := strings.Join(e.Tags, ", ")
			if len(tags) > 22 {
				tags = tags[:19] + "..."
			}
			desc := e.Description
			if len(desc) > 30 {
				desc = desc[:27] + "..."
			}
			e2eBadge := "-"
			switch e.E2EStatus {
			case "passed":
				e2eBadge = "✓"
			case "skipped":
				e2eBadge = "~"
			}
			marker := " "
			if e.Deprecated {
				marker = "⚠"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", marker, e.Name, e.LatestVersion, k, e2eBadge, tags, desc)
			count++
		}
		w.Flush()

		updatedAt := ""
		if latestUpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, latestUpdatedAt); err == nil {
				updatedAt = "  ·  Updated " + cmdutil.HumanDuration(time.Since(t)) + " ago"
			}
		}
		noun := "patterns"
		if count == 1 {
			noun = "pattern"
		}
		fmt.Printf("\n%d %s%s\n", count, noun, updatedAt)

		fmt.Printf("\nTo pull:\n  ork pull <name>:<version>\n")
		fmt.Printf("\nTo filter:\n  ork patterns --katalogs\n  ork patterns --motifs\n")
		fmt.Println()
		return nil
	},
}

// registryErrSummary extracts the last meaningful line from a verbose ORAS error.
func registryErrSummary(err error) string {
	msg := err.Error()
	if idx := strings.LastIndex(msg, ": "); idx >= 0 {
		return msg[idx+2:]
	}
	return msg
}

func init() {
	patternsCmd.Flags().StringP("tag", "t", "", "Filter by tag (e.g. database, stateful, security)")
	patternsCmd.Flags().BoolP("katalogs", "k", false, "Show only katalogs (kind: Katalog)")
	patternsCmd.Flags().BoolP("motifs", "m", false, "Show only motifs (kind: Motif)")
	cmdutil.RootCmd.AddCommand(patternsCmd)

	// Shadow global flags so they don't appear under `ork patterns`
	cmdutil.ShadowGlobalCommandFlags(patternsCmd, "file")
}
