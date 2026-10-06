//go:build !runtime && !gateway

package serve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/inrundev/inrun/cmd/cli/cmdutil"
	"github.com/inrundev/inrun/cmd/cli/gate"
	"github.com/inrundev/inrun/cmd/cli/testsuite"

	"github.com/inrundev/inrun/pkg/catalog"
	"github.com/inrundev/inrun/pkg/catalog/pipeline"
	"github.com/inrundev/inrun/pkg/gateway/api"
	"github.com/inrundev/inrun/pkg/intent"
	"github.com/inrundev/inrun/pkg/merger"
	"github.com/inrundev/inrun/pkg/registry/simulate"
	"github.com/inrundev/inrun/pkg/template"
	"github.com/inrundev/inrun/pkg/types"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// runCreateUpdateChain runs stages 1-6 of the play chain against an
// already-resolved intent map, printing each stage exactly like
// "inrun serve play" does. Shared with "inrun webhook play" — same engine,
// different front door: serve play reads its intent from a file, webhook
// play builds one from a source payload (push event, slash command, JSON
// body), then hands off here.
//
// source is stamped as the serve-source provenance annotation — "" for
// serve play (no caller identity beyond the token), or a webhook entry's
// own Name when called from webhook play, matching what ApplyTargetFields
// does for a real delivery. op is ServeOpCreate or ServeOpUpdate — webhook
// play always passes ServeOpCreate, since a delivered push/command has no
// --operation flag of its own.
func runCreateUpdateChain(k *catalog.Catalog, raw map[string]interface{}, tokenName, source string, op types.ServeOperation) (*unstructured.Unstructured, *types.CRDEntry, string, error) {
	target, _ := raw["target"].(string)
	if strings.TrimSpace(target) == "" {
		return nil, nil, "", fmt.Errorf(`"target" must be set in the intent`)
	}

	// Stage 1: Target resolution
	printStage(1, "Target resolution")
	resolution := k.LookupByTargetOrAlias(target)
	if resolution == nil {
		printStageError(fmt.Sprintf("unknown target %q\n  available: %s", target, strings.Join(k.AvailableTargets(), ", ")))
		return nil, nil, "", fmt.Errorf("target %q not found", target)
	}
	crd, alias := resolution.CRD, resolution.Alias
	printStageOK(fmt.Sprintf("kind=%s  target=%s  alias=%s", cmdutil.Bold(crd.Kind()), cmdutil.Green(crd.ServeTarget()), aliasOrNone(alias)))

	// Stage 2: Token check
	printStage(2, "Token check")
	allowed, denyReason := crd.TokenAllowedFor(alias, tokenName, string(op), "", types.ServeClassResources)
	if !allowed {
		msg := denyReason.Message(tokenName, string(op), crd.Kind(), "")
		printStageError(msg)
		return nil, nil, "", fmt.Errorf("token %q denied: %s", tokenName, msg)
	}
	printStageOK(fmt.Sprintf("token %s can %s on %s", cmdutil.Bold(tokenName), op, cmdutil.Bold(crd.Kind())))

	// Stage 3: CR construction
	printStage(3, "CR construction")
	notes := k.Notes
	obj, err := intent.Build(raw, crd, notes)
	if err != nil {
		printStageError(err.Error())
		return nil, nil, "", err
	}
	if obj.GetName() == "" {
		msg := "name is required — declare serve.name on the CRD or include \"name\" in the intent"
		printStageError(msg)
		return nil, nil, "", fmt.Errorf("%s", msg)
	}
	printStageOK(fmt.Sprintf("name=%s  namespace=%s", cmdutil.Bold(obj.GetName()), cmdutil.Bold(obj.GetNamespace())))

	// Stage 4: Provenance — stamp first, then print the CR so annotations are visible
	printStage(4, "Provenance annotations")
	api.InjectProvenanceAnnotations(obj, crd.ServeTarget(), alias, source)
	api.InjectServeIntentAnnotation(obj, raw)
	ann := obj.GetAnnotations()
	keys := make([]string, 0, len(ann))
	for k := range ann {
		if strings.HasPrefix(k, "inrun.") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		printStageDetail(fmt.Sprintf("%s: %s", cmdutil.Gray(k), cmdutil.Cyan(ann[k])))
	}
	printStageOK("provenance stamped — full CR:")
	printIndentedJSON(obj.Object)

	// Stage 5: Admission validation — same rules the webhook enforces at SSA time.
	// unique: operator is skipped (no live cluster); external: calls are skipped.
	printStage(5, "Admission validation")
	{
		resolver := template.NewResolverFromMap(obj.Object).WithUserNotes(k.Notes).WithRequest(raw)
		eval := resolver.TemplateEvaluator()
		r := gate.EvalAdmissionValidation(obj.Object, crd, resolver, eval)
		for _, v := range r.Violations {
			if v.Deny {
				printStageDetail(fmt.Sprintf("%s %s: %s", cmdutil.Red("deny"), cmdutil.Gray(v.Field), cmdutil.Red(v.Message)))
			} else {
				printStageDetail(fmt.Sprintf("%s %s: %s", cmdutil.Yellow("warn"), cmdutil.Gray(v.Field), cmdutil.Yellow(v.Message)))
			}
		}
		denied := r.Denied()
		if denied > 0 {
			denialTxt := "denials"
			violationTxt := "violations"
			if denied == 1 {
				denialTxt = "denial"
				violationTxt = "violation"
			}

			printStageError(fmt.Sprintf("%d %s — this CR would be rejected at admission", denied, denialTxt))
			return nil, nil, "", fmt.Errorf("admission denied: %d rule %s", denied, violationTxt)
		}
		warned := r.Warned()
		if warned > 0 {
			warningTxt := "warnings"
			if warned == 1 {
				warningTxt = "warning"
			}
			printStageOK(fmt.Sprintf("passed (%d %s)", warned, warningTxt))
		} else if r.Total > 0 {
			printStageOK("passed — no violations")
		} else {
			printStageOK("no validation rules declared")
		}

		m := gate.EvalAdmissionMutation(obj.Object, crd, resolver, eval)
		for _, p := range m.Previews {
			fromStr := cmdutil.Gray("(absent)")
			if p.Found {
				fromStr = cmdutil.Gray(p.From)
			}
			printStageDetail(fmt.Sprintf("%s %s: %s → %s  %s", cmdutil.Cyan("mut"), cmdutil.Gray(p.Field), fromStr, cmdutil.Cyan(fmt.Sprintf("%v", p.To)), cmdutil.Gray(p.MutType)))
		}
		if len(m.Previews) > 0 {
			mutTxt := "mutations"
			if len(m.Previews) == 1 {
				mutTxt = "mutation"
			}
			printStageOK(fmt.Sprintf("%d %s would be applied by the webhook", len(m.Previews), mutTxt))
		}
	}

	// Stage 6: Response payload
	printStage(6, "Response payload")
	payload := api.EvaluatePayload(obj.Object, crd, alias, notes)
	if payload == nil {
		printStageDetail(cmdutil.Gray("no serve.config.response declared — default CR response"))
	} else {
		printIndentedJSON(payload)
	}
	printStageOK("payload evaluated")

	return obj, crd, alias, nil
}

// playRunSimulate writes the built CR to a temp file and hands it to simulate.
// simulateConfig is the path to a simulate.yaml; empty means op-print mode with
// the same catalog play used. Shared by "inrun serve play --simulate" and
// "inrun webhook play --simulate" via playSimulate below.
func playRunSimulate(ctx context.Context, catalogFile string, obj *unstructured.Unstructured, simulateConfig string) error {
	b, err := yaml.Marshal(obj.Object)
	if err != nil {
		return fmt.Errorf("marshalling CR for simulate: %w", err)
	}
	tmp, err := os.CreateTemp("", "play-cr-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp CR file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp CR file: %w", err)
	}
	tmp.Close()

	fmt.Printf("%s  handoff to simulate\n", cmdutil.Cyan("─────────────────────────────────────────────────────────"))
	fmt.Println()

	if simulateConfig != "" {
		return runSimulateWithCR(ctx, simulateConfig, tmp.Name())
	}
	return testsuite.RunSimulate(ctx, catalogFile, tmp.Name(), testsuite.CliSimulateOptions{MaxCycles: 10, SkipExternal: true})
}

// runSimulateWithCR runs simulate using the spec file for catalog/cycles/expect
// but substitutes crFile (the play-built CR) instead of spec.cr.
// The expect: block is fully evaluated — this is assert mode, not op-print mode.
func runSimulateWithCR(ctx context.Context, specPath, crFile string) error {
	if abs, err := filepath.Abs(specPath); err == nil {
		specPath = abs
	}
	data, err := cmdutil.ReadLocal(specPath)
	if err != nil {
		return fmt.Errorf("reading simulate config %q: %w", specPath, err)
	}
	var doc types.Simulate
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing simulate config %q: %w", specPath, err)
	}
	if doc.Spec == nil {
		return fmt.Errorf("simulate config %q: missing spec", specPath)
	}

	catalogPath := cmdutil.JoinPath(specPath, doc.Spec.Catalog)
	cycles := doc.Spec.Cycles
	if cycles <= 0 {
		cycles = 10
	}
	opts := simulate.RunOptions{SkipExternal: doc.Spec.SkipExternal}

	m := merger.New(catalogPath)
	if err := m.Merge(); err != nil {
		return fmt.Errorf("merging Catalog: %w", err)
	}
	kat, err := pipeline.BuildExpanded(cmdutil.Kfg, m)
	if err != nil {
		return fmt.Errorf("parsing Catalog: %w", err)
	}

	crData, err := cmdutil.ReadLocal(crFile)
	if err != nil {
		return fmt.Errorf("reading CR: %w", err)
	}
	crs := cmdutil.ParseMultiDocCRs(crData)
	if len(crs) == 0 {
		return fmt.Errorf("no valid CR documents in %s", crFile)
	}

	var failed []string
	for _, name := range kat.CRDNames() {
		crdEntry, ok := kat.CRDEntry(name)
		if !ok {
			continue
		}
		in, ok := cmdutil.ResolveCRInputs(crs, crdEntry.APITypes.Kind)
		if !ok {
			if len(kat.CRDNames()) > 1 {
				fmt.Printf("  %s no CR found for %s — skipped\n\n", cmdutil.Dim("note:"), crdEntry.APITypes.Kind)
				continue
			}
			return fmt.Errorf("no CR found for CRD %q (kind: %s)", name, crdEntry.APITypes.Kind)
		}
		crdOpts := opts
		crdOpts.Peers = in.Peers
		crdOpts.ExistingInstances = in.Existing
		expect := simulate.ExpectForCRD(doc.Spec.Expect, name)
		if err := testsuite.SimulateOne(ctx, kat, name, in.CR, cycles, crdOpts, testsuite.CliSimulateOptions{}, nil, expect); err != nil {
			failed = append(failed, name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("simulate failed for: %s", strings.Join(failed, ", "))
	}
	return nil
}

// RunIntentPlay builds a Catalog from catalogPath, runs the play chain
// against intentFile without printing stage output, and returns the resolved
// target name on success.
func RunIntentPlay(catalogPath, intentFile string) (string, error) {
	m := merger.New(catalogPath)
	if err := m.Merge(); err != nil {
		return "", fmt.Errorf("merging catalog: %w", err)
	}
	k, err := pipeline.BuildExpanded(cmdutil.Kfg, m)
	if err != nil {
		return "", fmt.Errorf("building catalog: %w", err)
	}

	raw, err := readIntentFile(intentFile)
	if err != nil {
		return "", fmt.Errorf("reading intent file: %w", err)
	}

	target, _ := raw["target"].(string)
	if strings.TrimSpace(target) == "" {
		return "", fmt.Errorf(`intent file must declare a "target"`)
	}

	resolution := k.LookupByTargetOrAlias(target)
	if resolution == nil {
		return target, fmt.Errorf("unknown target %q — available: %s", target, strings.Join(k.AvailableTargets(), ", "))
	}
	crd, alias := resolution.CRD, resolution.Alias

	if tokenName, _ := raw["token"].(string); tokenName != "" {
		allowed, denyReason := crd.TokenAllowedFor(alias, tokenName, string(types.ServeOpCreate), "", types.ServeClassResources)
		if !allowed {
			return target, fmt.Errorf("token %q denied: %s", tokenName, denyReason.Message(tokenName, string(types.ServeOpCreate), crd.Kind(), ""))
		}
	} else {
		return target, fmt.Errorf("intent file must declare a 'token' — token: <name>")
	}

	obj, err := intent.Build(raw, crd, k.Notes)
	if err != nil {
		return target, fmt.Errorf("CR construction: %w", err)
	}

	resolver := template.NewResolverFromMap(obj.Object).WithUserNotes(k.Notes).WithRequest(raw)
	eval := resolver.TemplateEvaluator()
	r := gate.EvalAdmissionValidation(obj.Object, crd, resolver, eval)
	denied := r.Denied()
	violationTxt := "violations"
	if denied > 0 {
		if denied == 1 {
			violationTxt = "violation"
		}
		return target, fmt.Errorf("admission denied: %d rule %s", denied, violationTxt)
	}

	return target, nil
}

// webhookSimulate carries the --simulate handoff arguments through
// "inrun webhook play" to wherever a play chain successfully builds a CR —
// same handoff "inrun serve play --simulate" does (playRunSimulate), just
// reached from a different front door, and potentially more than once per
// run: a GitHub/GitLab push can match several files, each its own build.
type webhookSimulate struct {
	ctx         context.Context
	catalogFile string
	enabled     bool
	config      string
}

// playSimulate hands obj to inrun simulate when sim.enabled — a no-op
// otherwise. Called after every successful runCreateUpdateChain in
// "inrun webhook play".
func playSimulate(sim webhookSimulate, obj *unstructured.Unstructured) error {
	if !sim.enabled {
		return nil
	}
	return playRunSimulate(sim.ctx, sim.catalogFile, obj, sim.config)
}
