package plan

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	"reflect"
)

// KatalogDiff holds the structured difference between two Katalogs.
type KatalogDiff struct {
	AddedCRDs   []string
	RemovedCRDs []string
	ChangedCRDs []CRDDiff
}

type CRDDiff struct {
	Name    string
	Changes []FieldChange
}

type FieldChange struct {
	Path string
	From string // "" means new
	To   string // "" means removed
}

func (d *KatalogDiff) Empty() bool {
	return len(d.AddedCRDs) == 0 &&
		len(d.RemovedCRDs) == 0 &&
		len(d.ChangedCRDs) == 0
}

func (d *KatalogDiff) Print() {
	for _, name := range d.AddedCRDs {
		fmt.Printf("  %s CRD '%s'  (new)\n", utils.Green("+"), name)
	}
	for _, name := range d.RemovedCRDs {
		fmt.Printf("  %s CRD '%s'  (removed)\n", utils.Red("-"), name)
	}
	for _, crd := range d.ChangedCRDs {
		fmt.Printf("\n  CRD '%s':\n", crd.Name)
		for _, ch := range crd.Changes {
			if ch.From == "" {
				fmt.Printf("    %s %s  (new)\n", utils.Green("+"), ch.Path)
			} else if ch.To == "" {
				fmt.Printf("    %s %s  (removed)\n", utils.Red("-"), ch.Path)
			} else {
				fmt.Printf("    %s %s:  %s → %s\n",
					utils.Yellow("~"), ch.Path,
					utils.Red(ch.From), utils.Green(ch.To))
			}
		}
	}
	fmt.Println()
}

// ComputeKatalogDiff computes the structured diff between two Katalogs.
func ComputeKatalogDiff(from, to *katalog.Katalog) *KatalogDiff {
	diff := &KatalogDiff{}

	fromCRDs := from.CRDNames()
	toCRDs := to.CRDNames()

	fromSet := makeSet(fromCRDs)
	toSet := makeSet(toCRDs)

	for _, name := range toCRDs {
		if !fromSet[name] {
			diff.AddedCRDs = append(diff.AddedCRDs, name)
		}
	}
	for _, name := range fromCRDs {
		if !toSet[name] {
			diff.RemovedCRDs = append(diff.RemovedCRDs, name)
		}
	}

	// Changed CRDs — compare fields
	for _, name := range fromCRDs {
		if !toSet[name] {
			continue
		}
		fromCRD, _ := from.CRDEntry(name)
		toCRD, _ := to.CRDEntry(name)
		changes := diffCRDEntry(name, &fromCRD, &toCRD)
		if len(changes) > 0 {
			diff.ChangedCRDs = append(diff.ChangedCRDs, CRDDiff{
				Name:    name,
				Changes: changes,
			})
		}
	}

	return diff
}

func diffCRDEntry(_ string, from, to *orktypes.CRDEntry) []FieldChange {
	var changes []FieldChange

	// metadata
	if from.Name != to.Name {
		changes = append(changes, FieldChange{
			Path: "name",
			From: fmt.Sprint(from.Name),
			To:   fmt.Sprint(to.Name),
		})
	}
	if from.Namespace != to.Namespace {
		changes = append(changes, FieldChange{
			Path: "namespace",
			From: fmt.Sprint(from.Namespace),
			To:   fmt.Sprint(to.Namespace),
		})
	}
	if from.Description != to.Description {
		changes = append(changes, FieldChange{
			Path: "description",
			From: fmt.Sprint(from.Description),
			To:   fmt.Sprint(to.Description),
		})
	}

	// apitypes
	if !reflect.DeepEqual(from.APITypes, to.APITypes) {
		changes = append(changes, FieldChange{
			Path: "apiTypes",
			From: fmt.Sprint(from.APITypes),
			To:   fmt.Sprint(to.APITypes),
		})
	}

	// spec
	if from.SetWorkers(0) != to.SetWorkers(0) {
		changes = append(changes, FieldChange{
			Path: "workers",
			From: fmt.Sprint(from.SetWorkers(0)),
			To:   fmt.Sprint(to.SetWorkers(0)),
		})
	}
	if from.SetResync(0) != to.SetResync(0) {
		changes = append(changes, FieldChange{
			Path: "resync",
			From: from.SetResync(0).String(),
			To:   to.SetResync(0).String(),
		})
	}
	if from.CRDFile != to.CRDFile {
		changes = append(changes, FieldChange{
			Path: "crdFile",
			From: from.CRDFile,
			To:   to.CRDFile,
		})
	}
	if from.SetQueueDepth(0) != to.SetQueueDepth(0) {
		changes = append(changes, FieldChange{
			Path: "queue",
			From: fmt.Sprint(from.SetQueueDepth(0)),
			To:   fmt.Sprint(to.SetQueueDepth(0)),
		})
	}
	if !reflect.DeepEqual(from.DependsOn, to.DependsOn) {
		changes = append(changes, FieldChange{
			Path: "dependsOn",
			From: fmt.Sprint(from.DependsOn),
			To:   fmt.Sprint(to.DependsOn),
		})
	}

	// namespace restrictions
	if !reflect.DeepEqual(from.AllAllowedNamespaces(), to.AllAllowedNamespaces()) {
		changes = append(changes, FieldChange{
			Path: "allowedNamespaces",
			From: fmt.Sprint(from.AllAllowedNamespaces()),
			To:   fmt.Sprint(to.AllAllowedNamespaces()),
		})
	}
	if !reflect.DeepEqual(from.AllRestrictedNamespaces(), to.AllRestrictedNamespaces()) {
		changes = append(changes, FieldChange{
			Path: "restrictedNamespaces",
			From: fmt.Sprint(from.AllRestrictedNamespaces()),
			To:   fmt.Sprint(to.AllRestrictedNamespaces()),
		})
	}

	// Selectors
	if !reflect.DeepEqual(from.LabelSelector, to.LabelSelector) {
		changes = append(changes, FieldChange{
			Path: "labelSelector",
			From: fmt.Sprint(from.LabelSelector),
			To:   fmt.Sprint(to.LabelSelector),
		})
	}
	if !reflect.DeepEqual(from.FieldSelector, to.FieldSelector) {
		changes = append(changes, FieldChange{
			Path: "fieldSelector",
			From: fmt.Sprint(from.FieldSelector),
			To:   fmt.Sprint(to.FieldSelector),
		})
	}

	// validation, mutation, conversion
	if !reflect.DeepEqual(from.EffectiveValidation(), to.EffectiveValidation()) {
		changes = append(changes, FieldChange{
			Path: "validation",
			From: fmt.Sprint(from.EffectiveValidation()),
			To:   fmt.Sprint(to.EffectiveValidation()),
		})
	}
	if !reflect.DeepEqual(from.EffectiveMutation(), to.EffectiveMutation()) {
		changes = append(changes, FieldChange{
			Path: "mutation",
			From: fmt.Sprint(from.EffectiveMutation()),
			To:   fmt.Sprint(to.EffectiveMutation()),
		})
	}
	if !reflect.DeepEqual(from.EffectiveConversion(), to.EffectiveConversion()) {
		changes = append(changes, FieldChange{
			Path: "conversion",
			From: fmt.Sprint(from.EffectiveConversion()),
			To:   fmt.Sprint(to.EffectiveConversion()),
		})
	}

	// Normalize
	if !reflect.DeepEqual(from.EffectiveNormalize(), to.EffectiveNormalize()) {
		changes = append(changes, FieldChange{
			Path: "normalize",
			From: fmt.Sprint(from.EffectiveNormalize()),
			To:   fmt.Sprint(to.EffectiveNormalize()),
		})
	}

	// operatorBox resource counts
	if from.OperatorBox.EffectiveOnCreate() != nil && to.OperatorBox.EffectiveOnCreate() != nil {
		fromCreate := resourceCounts(from.OperatorBox.EffectiveOnCreate())
		toCreate := resourceCounts(to.OperatorBox.EffectiveOnCreate())
		for _, change := range diffResourceCounts("operatorBox.onCreate", fromCreate, toCreate) {
			changes = append(changes, change)
		}
	}
	if from.OperatorBox.EffectiveOnReconcile() != nil && to.OperatorBox.EffectiveOnReconcile() != nil {
		fromReconcile := resourceCounts(from.OperatorBox.EffectiveOnReconcile())
		toReconcile := resourceCounts(to.OperatorBox.EffectiveOnReconcile())
		for _, change := range diffResourceCounts("operatorBox.onReconcile", fromReconcile, toReconcile) {
			changes = append(changes, change)
		}
	}

	return changes
}

func resourceCounts(ht *orktypes.HookTemplates) map[string]int {
	if ht == nil {
		return nil
	}
	return map[string]int{
		"deployments":            len(ht.Deployments),
		"services":               len(ht.Services),
		"secrets":                len(ht.Secrets),
		"configMaps":             len(ht.ConfigMaps),
		"serviceAccounts":        len(ht.ServiceAccounts),
		"statefulSets":           len(ht.StatefulSets),
		"ingresses":              len(ht.Ingresses),
		"jobs":                   len(ht.Jobs),
		"cronJobs":               len(ht.CronJobs),
		"persistentVolumes":      len(ht.PersistentVolumes),
		"persistentVolumeClaims": len(ht.PersistentVolumeClaims),
		"hpa":                    len(ht.HorizontalPodAutoscalers),
		"pdb":                    len(ht.PodDisruptionBudgets),
		"namespaces":             len(ht.Namespaces),
		"roles":                  len(ht.Roles),
		"roleBindings":           len(ht.RoleBindings),
		"custom":                 len(ht.CustomResource),
		"replicaSets":            len(ht.ReplicaSets),
		"pods":                   len(ht.Pods),
	}
}

func diffResourceCounts(prefix string, from, to map[string]int) []FieldChange {
	var changes []FieldChange
	for key, toVal := range to {
		fromVal := from[key]
		if fromVal != toVal {
			changes = append(changes, FieldChange{
				Path: strings.Join([]string{prefix, key}, "."),
				From: fmt.Sprint(fromVal),
				To:   fmt.Sprint(toVal),
			})
		}
	}
	return changes
}

func makeSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, v := range items {
		s[v] = true
	}
	return s
}
