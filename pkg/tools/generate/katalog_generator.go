// pkg/generate/katalog_generator.go
//
// KatalogScaffold generates a starter katalog.yaml for a new operator.
//
// Three reconcile modes:
//
//	dynamic (default) — operatorBox.default: true; commented onCreate / onReconcile / onDelete
//	    template blocks are included so the user can see the available structure.
//
//	typed + hooks  — mode: typed, commented hooks declaration, default: true.
//	    The user writes a Go hook function and registers it in the Katalog.
//
//	typed + constructor — mode: typed, operatorBox.default: false, commented constructor.
//	    The user owns the entire reconcile loop; Orkestra calls their constructor.
//
// Optional sections (security, notification) are injected after the metadata
// block when the corresponding flag is set. They are independent of the
// reconcile mode and may be combined freely.
//
// The generated file is pure YAML — all conditional logic is resolved at
// generation time by the Go template. No template syntax leaks into the output.
package generate

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// KatalogScaffoldOptions controls what the scaffold generates.
//
// At most one of AddHook, AddConstructor, or Typed may be true.
// All other options are additive and may be combined with any mode.
type KatalogScaffoldOptions struct {
	// AddHook generates a typed Katalog with a commented hooks declaration.
	// Mutually exclusive with AddConstructor and Typed.
	AddHook bool

	// AddConstructor generates a typed Katalog with a commented constructor
	// declaration and operatorBox.default: false.
	// Mutually exclusive with AddHook and Typed.
	AddConstructor bool

	// Typed generates a typed Katalog with both hooks and constructor sections
	// commented. A warning is printed to stderr — the user must uncomment one
	// and delete the other before running. Mutually exclusive with AddHook and
	// AddConstructor.
	Typed bool

	// AddSecurity appends a security block (namespace + deletion protection).
	AddSecurity bool

	// AddNotification appends a notification block with example team entries.
	AddNotification bool

	// OutputFile is the destination path. Defaults to "katalog.yaml".
	OutputFile string
}

// Validate returns a descriptive error when mutually exclusive flags are combined.
func (o KatalogScaffoldOptions) Validate() error {
	typeFlags := 0
	if o.AddHook {
		typeFlags++
	}
	if o.AddConstructor {
		typeFlags++
	}
	if o.Typed {
		typeFlags++
	}
	if typeFlags > 1 {
		return errors.New(
			"--add-hook, --add-constructor, and --typed are mutually exclusive; " +
				"use --typed to get both sections commented so you can choose one",
		)
	}
	return nil
}

// katalogTemplateData is the internal model passed to the YAML template.
// All conditional decisions are made before rendering; the template is a
// plain substitution engine with no business logic inside it.
type katalogTemplateData struct {
	// FlagSuffix appears in the generated-by comment so the file is self-documenting.
	FlagSuffix string

	// IsTyped is true for any of the three typed-mode flags.
	IsTyped bool

	// ShowHooks includes the commented hooks declaration.
	ShowHooks bool

	// ShowConstructor includes the commented constructor declaration.
	ShowConstructor bool

	// DefaultFalse sets operatorBox.default: false (constructor mode only).
	DefaultFalse bool

	// AddSecurity, AddNotification control optional top-level blocks.
	AddSecurity     bool
	AddNotification bool

	// Timestamp is written into the generated-by comment.
	Timestamp string
}

// KatalogScaffold renders a starter katalog.yaml according to opts and writes
// it to opts.OutputFile (default "katalog.yaml"). The rendered YAML is also
// returned as a string so callers can use it for dry-run display or testing.
//
// When opts.Typed is true, a warning is printed to stderr explaining that the
// user must uncomment exactly one of the two generated sections.
func KatalogScaffold(opts KatalogScaffoldOptions) (string, error) {
	if err := opts.Validate(); err != nil {
		return "", err
	}

	data := katalogTemplateData{
		AddSecurity:     opts.AddSecurity,
		AddNotification: opts.AddNotification,
		Timestamp:       time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}

	switch {
	case opts.AddHook:
		data.FlagSuffix = " --add-hook"
		data.IsTyped = true
		data.ShowHooks = true
	case opts.AddConstructor:
		data.FlagSuffix = " --add-constructor"
		data.IsTyped = true
		data.ShowConstructor = true
		data.DefaultFalse = true
	case opts.Typed:
		data.FlagSuffix = " --typed"
		data.IsTyped = true
		data.ShowHooks = true
		data.ShowConstructor = true
	}
	if opts.AddSecurity {
		data.FlagSuffix += " --add-security"
	}
	if opts.AddNotification {
		data.FlagSuffix += " --add-notification"
	}

	var buf bytes.Buffer
	if err := katalogTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering katalog scaffold: %w", err)
	}
	out := buf.String()

	if opts.Typed {
		fmt.Fprintln(os.Stderr,
			"WARNING: Typed mode requires either 'hooks' or 'constructor' "+
				"to be uncommented and properly configured.")
	}

	dest := opts.OutputFile
	if dest == "" {
		dest = "katalog.yaml"
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("creating directory for %s: %w", dest, err)
	}
	if err := os.WriteFile(dest, []byte(out), 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", dest, err)
	}
	return out, nil
}

// katalogTmpl is the single template that covers all scaffold variants.
// It uses only built-in template actions (if / not / eq) — no FuncMap needed.
// Template syntax appearing inside YAML comments is escaped via {{ "{{" }}.
var katalogTmpl = template.Must(template.New("katalog-scaffold").Parse(
	`# Generated by "ork generate katalog{{ .FlagSuffix }}" on {{ .Timestamp }}
# https://orkestra.sh/docs/reference/schema/katalog/
# Edit every TODO placeholder before running "ork run".
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: my-operator
  author: TODO
  version: 0.1.0
  description: TODO
{{ if .AddSecurity -}}
security:
  namespaceProtection:
    enabled: false
    # allowedNamespaces:
    #   - default
    #   - production
  deletionProtection:
    enabled: false

{{ end -}}
{{ if .AddNotification -}}
notification:
  enabled: true
  defaults:
    interval: 15m
    slackWebhookUrl: ${SLACK_ORG_WEBHOOK}
  teams:
    platform:
      email:
        - platform@myorg.com
      slack:
        - "#platform-alerts"
      slackWebhookUrl: ${SLACK_PLATFORM_WEBHOOK}
      interval: 5m
    oncall:
      slack:
        - "#oncall"
      interval: 1m

{{ end -}}
spec:
  crds:
    my-resource:
      apiTypes:
        group: TODO
        version: v1alpha1
        kind: TODO
        plural: TODO
{{ if .IsTyped }}        object: TODO
        objectList: TODO
        location: TODO # github.com/myorg/my-operator/api/v1alpha1
      mode: typed
{{ end }}      operatorBox:
        reconciler:
          default: {{ if .DefaultFalse }}false{{ else }}true{{ end }}
          workers: 3
          resync: 30s
          queue:
            maxDepth: 100
{{ if .ShowHooks }}
          # hooks:
          #   # Package exporting the hook factory function.
          #   location: github.com/myorg/my-operator/hooks
          #   # Function signature: func() domain.AnyReconcileHooks
          #   # Return:             domain.ReconcileHooks[*MyKind]{OnReconcile: ...}
          #   function: MyResourceHooks
          #   alias: myhook
{{ end -}}
{{ if .ShowConstructor }}
          # constructor:
          #   # Package exporting the reconciler constructor.
          #   location: github.com/myorg/my-operator/reconciler
          #   # Function signature:
          #   #   func(*kubeclient.Kubeclient, cache.SharedIndexInformer, *event.Event) domain.Reconciler
          #   function: NewMyResourceReconciler
          #   alias: myrec
{{ end -}}
{{ if not .IsTyped }}
        # Declarative template blocks — uncomment and fill in what you need.
        # onCreate:
        #   deployments:
        #     - name: {{ "{{" }} .metadata.name {{ "}}" }}
        #       image: nginx:latest
        #       replicas: 1
        #   services:
        #     - name: {{ "{{" }} .metadata.name {{ "}}" }}-svc
        #       port: "80"
        #       targetPort: "80"
        #       type: ClusterIP
        # onReconcile: {}
        # onDelete: {}

        # Declare status fields written after each successful reconcile.
        # status:
        #   fields:
        #     - path: phase
        #       value: "Running"
{{ end -}}
`,
))
