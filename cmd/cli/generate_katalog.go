// cmd/cli/generate_katalog.go
//
// "ork generate katalog" scaffolds a starter katalog.yaml.
//
// All reconcile-mode flags are mutually exclusive:
//
//	(none)             → dynamic mode  — declarative templates, no Go code required
//	--add-hook         → typed mode    — hooks section; user writes ReconcileHooks[*T]
//	--add-constructor  → typed mode    — constructor section; user owns the reconcile loop
//	--typed            → typed mode    — both sections commented; warning printed to stderr
//
// Optional sections may be combined with any mode:
//
//	--add-security          — namespace + deletion protection block
//	--add-notification      — notification / alerting block

//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"log"

	"github.com/orkspace/orkestra/pkg/tools/generate"
	"github.com/spf13/cobra"
)

var generateKatalogCmd = &cobra.Command{
	Use:   "katalog",
	Short: "Scaffold a starter katalog.yaml for a new operator",
	Long: `Generates a katalog.yaml with sensible defaults, commented optional blocks,
and TODO placeholders. All generated fields are valid YAML — no further tooling is
required before editing.

Reconcile mode (choose at most one):
  (none)              Dynamic mode: declarative templates, no Go code needed.
  --add-hook          Typed mode: includes a commented 'hooks' declaration.
  --add-constructor   Typed mode: includes a commented 'constructor' declaration.
  --typed             Typed mode: includes both 'hooks' and 'constructor' commented.
                      A warning is printed — uncomment exactly one.

Optional sections (may be combined with any mode):
  --add-security        Namespace and deletion-protection block.
  --add-notification    Notification / alerting block with example teams.

Examples:
  ork generate katalog
  ork generate katalog --add-hook -o database-katalog.yaml
  ork generate katalog --add-constructor
  ork generate katalog --typed --add-security
  ork generate katalog --add-notification -o ops-katalog.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addHook, _ := cmd.Flags().GetBool("add-hook")
		addConstructor, _ := cmd.Flags().GetBool("add-constructor")
		typed, _ := cmd.Flags().GetBool("typed")
		addSecurity, _ := cmd.Flags().GetBool("add-security")
		addNotification, _ := cmd.Flags().GetBool("add-notification")
		outputFile, _ := cmd.Flags().GetString("output")

		opts := generate.KatalogScaffoldOptions{
			AddHook:         addHook,
			AddConstructor:  addConstructor,
			Typed:           typed,
			AddSecurity:     addSecurity,
			AddNotification: addNotification,
			OutputFile:      outputFile,
		}

		// Validate before calling the generator so errors surface before any I/O.
		if err := opts.Validate(); err != nil {
			return err
		}

		dest := outputFile
		if dest == "" {
			dest = "katalog.yaml"
		}

		log.Printf("generating katalog scaffold → %s\n", dest)

		if _, err := generate.KatalogScaffold(opts); err != nil {
			return fmt.Errorf("generate katalog: %w", err)
		}

		log.Printf("katalog scaffold written to %s\n", dest)
		return nil
	},
}

func init() {
	generateKatalogCmd.Flags().Bool("add-hook", false,
		"Typed mode: include a commented hooks declaration (mutually exclusive with --add-constructor and --typed)")
	generateKatalogCmd.Flags().Bool("add-constructor", false,
		"Typed mode: include a commented constructor declaration with default: false (mutually exclusive with --add-hook and --typed)")
	generateKatalogCmd.Flags().Bool("typed", false,
		"Typed mode: include both hooks and constructor sections commented; prints a warning")
	generateKatalogCmd.Flags().Bool("add-security", false,
		"Include a security block (namespace protection + deletion protection)")
	generateKatalogCmd.Flags().Bool("add-notification", false,
		"Include a notification block with example team entries")
	generateKatalogCmd.Flags().StringP("output", "o", "",
		`Output file path (default "katalog.yaml")`)
}
