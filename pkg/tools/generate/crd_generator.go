// pkg/generat/crd_generator.go
//
// CRD generator — derives a CustomResourceDefinition from a Katalog.
//
// The Katalog is the single source of truth. Everything needed to produce
// a valid CRD is already declared:
//
//	apiTypes        → group, version, kind, plural, scope
//	validation      → required fields (deny rules with operator: exists)
//	mutation        → optional fields with defaults + type inference
//	template exprs  → additional spec fields referenced as {{ .spec.* }}
//	status.fields   → status subresource schema + printer columns
//	conversion      → webhook config (when conversion paths are declared)
//
// Usage:
//
//	gen := generator.NewCRDGenerator(katalogEntry)
//	crd, err := gen.Generate()
//	yaml.NewEncoder(os.Stdout).Encode(crd)
//
// CLI:
//
//	ork generate crd --file katalog.yaml -o crd.yaml
package generate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CRDGenerator produces a CRD from a CRDEntry.
type CRDGenerator struct {
	crd orktypes.CRDEntry
}

// NewCRDGenerator creates a generator for one CRD entry.
func NewCRDGenerator(crd orktypes.CRDEntry) *CRDGenerator {
	return &CRDGenerator{crd: crd}
}

// Generate produces the complete CRD object.
func (g *CRDGenerator) Generate() (*apiextv1.CustomResourceDefinition, error) {
	spec := g.buildSpec()

	name := fmt.Sprintf("%s.%s",
		strings.ToLower(g.crd.APITypes.Plural),
		g.crd.APITypes.Group,
	)

	return &apiextv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}, nil
}

func (g *CRDGenerator) buildSpec() apiextv1.CustomResourceDefinitionSpec {
	scope := apiextv1.NamespaceScoped
	if g.crd.Namespaced != nil && !*g.crd.Namespaced {
		scope = apiextv1.ClusterScoped
	}

	version := g.buildVersion()
	spec := apiextv1.CustomResourceDefinitionSpec{
		Group: g.crd.APITypes.Group,
		Scope: scope,
		Names: apiextv1.CustomResourceDefinitionNames{
			Plural:   strings.ToLower(g.crd.APITypes.Plural),
			Singular: strings.ToLower(g.crd.APITypes.Kind),
			Kind:     g.crd.APITypes.Kind,
		},
		Versions: []apiextv1.CustomResourceDefinitionVersion{version},
	}

	// Conversion webhook — when conversion paths are declared
	if conv := g.crd.EffectiveConversion(); conv != nil && len(conv.Paths) > 0 {
		storageVersion := g.crd.APITypes.Version
		if conv.StorageVersion != "" {
			storageVersion = conv.StorageVersion
		}

		spec.Conversion = &apiextv1.CustomResourceConversion{
			Strategy: apiextv1.WebhookConverter,
			Webhook: &apiextv1.WebhookConversion{
				ClientConfig: &apiextv1.WebhookClientConfig{
					Service: &apiextv1.ServiceReference{
						Name:      "orkestra",
						Namespace: "orkestra-system",
						Path:      strPtr("/convert"),
						Port:      int32Ptr(8443),
					},
					CABundle: nil, // caBundle: filled in after cert generation
				},
				ConversionReviewVersions: []string{storageVersion},
			},
		}
	}

	return spec
}

func (g *CRDGenerator) buildVersion() apiextv1.CustomResourceDefinitionVersion {
	specProps := g.inferSpecProperties()
	statusProps := g.inferStatusProperties()
	printerCols := g.buildPrinterColumns()

	schema := &apiextv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextv1.JSONSchemaProps{
			"spec":   {Type: "object", Properties: specProps, Required: g.requiredFields()},
			"status": {Type: "object", XPreserveUnknownFields: boolPtr(true), Properties: statusProps},
		},
	}

	storageVersion := g.crd.APITypes.Version
	if conv := g.crd.EffectiveConversion(); conv != nil && conv.StorageVersion != "" {
		storageVersion = conv.StorageVersion
	}

	storage := false
	if storageVersion == g.crd.APITypes.Version {
		storage = true
	}

	return apiextv1.CustomResourceDefinitionVersion{
		Name:    g.crd.APITypes.Version,
		Served:  true,
		Storage: storage,
		Subresources: &apiextv1.CustomResourceSubresources{
			Status: &apiextv1.CustomResourceSubresourceStatus{},
		},
		AdditionalPrinterColumns: printerCols,
		Schema: &apiextv1.CustomResourceValidation{
			OpenAPIV3Schema: schema,
		},
	}
}

// inferSpecProperties derives the spec schema from three sources:
//  1. Validation deny rules (required fields)
//  2. Mutation default rules (optional fields with type from default value)
//  3. Template expressions {{ .spec.* }} in all template blocks
func (g *CRDGenerator) inferSpecProperties() map[string]apiextv1.JSONSchemaProps {
	fields := make(map[string]fieldInfo)

	// Source 1: validation rules
	if v := g.crd.EffectiveValidation(); v != nil {
		for _, rule := range v.Rules {
			path := trimSpecPrefix(rule.Field)
			if path == "" || strings.Contains(path, ".") {
				continue // skip status.* and nested fields at spec level
			}
			info := fields[path]
			info.name = path
			if rule.Action == "deny" && string(rule.Operator) == "exists" {
				info.required = true
			}
			if info.typ == "" {
				info.typ = "string" // default
			}
			if rule.Message != "" {
				info.description = rule.Message
			}
			fields[path] = info
		}
	}

	// Source 2: mutation defaults — type inferred from default value
	if m := g.crd.EffectiveMutation(); m != nil {
		for _, rule := range m.Rules {
			path := trimSpecPrefix(rule.Field)
			if path == "" || strings.Contains(path, ".") {
				continue
			}
			info := fields[path]
			info.name = path
			if rule.Default != nil {
				info.typ = inferTypeFromValue(rule.Default)
				info.defaultVal = rule.Default
			}
			fields[path] = info
		}
	}

	// Source 3: template expressions in all declared templates
	for _, tmplPath := range g.extractTemplateSpecFields() {
		if _, exists := fields[tmplPath]; !exists {
			fields[tmplPath] = fieldInfo{name: tmplPath, typ: "string"}
		}
	}

	// Build the properties map
	props := make(map[string]apiextv1.JSONSchemaProps, len(fields))
	for name, info := range fields {
		prop := apiextv1.JSONSchemaProps{
			Type: info.typ,
		}
		if info.description != "" {
			prop.Description = info.description
		}
		if info.defaultVal != nil {
			raw, _ := toRawExtension(info.defaultVal)
			prop.Default = raw
		}
		props[name] = prop
	}

	return props
}

// inferStatusProperties derives the status schema from status.fields declarations.
func (g *CRDGenerator) inferStatusProperties() map[string]apiextv1.JSONSchemaProps {
	if g.crd.Box().EffectiveStatus() == nil {
		return nil
	}

	props := map[string]apiextv1.JSONSchemaProps{
		"conditions": {
			Type: "array",
			Items: &apiextv1.JSONSchemaPropsOrArray{
				Schema: &apiextv1.JSONSchemaProps{
					Type:                   "object",
					XPreserveUnknownFields: boolPtr(true),
				},
			},
		},
		"observedGeneration": {Type: "integer"},
	}

	for _, field := range g.crd.Box().EffectiveStatus().Fields {
		path := field.Path
		if strings.Contains(path, ".") {
			continue // nested — handled by x-kubernetes-preserve-unknown-fields
		}
		if _, exists := props[path]; !exists {
			props[path] = apiextv1.JSONSchemaProps{Type: "string"}
		}
	}

	return props
}

// requiredFields returns spec field names that have deny rules with operator: exists.
func (g *CRDGenerator) requiredFields() []string {
	v := g.crd.EffectiveValidation()
	if v == nil {
		return nil
	}
	var required []string
	seen := map[string]bool{}
	for _, rule := range v.Rules {
		path := trimSpecPrefix(rule.Field)
		if path == "" || strings.Contains(path, ".") {
			continue
		}
		if rule.Action == "deny" && string(rule.Operator) == "exists" && !seen[path] {
			required = append(required, path)
			seen[path] = true
		}
	}
	sort.Strings(required)
	return required
}

// buildPrinterColumns generates additionalPrinterColumns from status.fields.
// The first status field becomes the primary column. phase is always first if present.
func (g *CRDGenerator) buildPrinterColumns() []apiextv1.CustomResourceColumnDefinition {
	cols := []apiextv1.CustomResourceColumnDefinition{
		{
			Name:     "Age",
			Type:     "date",
			JSONPath: ".metadata.creationTimestamp",
		},
	}

	if g.crd.Box().EffectiveStatus() == nil {
		return cols
	}

	seen := map[string]bool{"age": true}
	var statusCols []apiextv1.CustomResourceColumnDefinition

	// phase first
	for _, field := range g.crd.Box().EffectiveStatus().Fields {
		if field.Path == "phase" && !seen["phase"] {
			statusCols = append([]apiextv1.CustomResourceColumnDefinition{
				{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
			}, statusCols...)
			seen["phase"] = true
		}
	}

	// other simple fields (max 3 additional columns)
	count := 0
	for _, field := range g.crd.Box().EffectiveStatus().Fields {
		if strings.Contains(field.Path, ".") || seen[field.Path] || count >= 3 {
			continue
		}
		statusCols = append(statusCols, apiextv1.CustomResourceColumnDefinition{
			Name:     toTitle(field.Path),
			Type:     "string",
			JSONPath: ".status." + field.Path,
		})
		seen[field.Path] = true
		count++
	}

	return append(statusCols, cols...)
}

// extractTemplateSpecFields parses all template expressions across
// all declared template blocks and extracts top-level .spec.* references.
func (g *CRDGenerator) extractTemplateSpecFields() []string {
	// Collect all raw template strings from the reconciler config
	var templates []string
	op := g.crd.OperatorBox

	collectFromTemplates := func(t *orktypes.HookTemplates) {
		if t == nil {
			return
		}
		for _, d := range t.Deployments {
			templates = append(templates, d.Image, d.Name, d.Namespace)
		}
		for _, s := range t.Services {
			templates = append(templates, s.Name, s.Namespace)
		}
		for _, j := range t.Jobs {
			templates = append(templates, j.Image, j.Name)
			templates = append(templates, j.Command...)
		}
		for _, cj := range t.CronJobs {
			templates = append(templates, cj.Image, cj.Name, cj.Schedule)
		}
		for _, cm := range t.ConfigMaps {
			templates = append(templates, cm.Name)
		}
	}

	collectFromTemplates(op.EffectiveOnCreate())
	collectFromTemplates(op.EffectiveOnReconcile())

	// Also collect from provider block fields
	if r := op.Reconcile; r != nil {
		for _, block := range r.ProviderBlocks {
			for _, decl := range block.Declarations {
				for _, v := range decl.Fields {
					templates = append(templates, v)
				}
			}
		}
	}

	// Also collect from status fields
	if s := op.EffectiveStatus(); s != nil {
		for _, f := range s.Fields {
			templates = append(templates, f.Value)
		}
	}

	return extractSpecPaths(templates)
}

// ─────────────────────────────────────────────────────────────────────────────
// CR generator
// ─────────────────────────────────────────────────────────────────────────────

// CRGenerator produces an example CR from a CRDEntry.
type CRGenerator struct {
	crd orktypes.CRDEntry
}

// NewCRGenerator creates a CR generator for one CRD entry.
func NewCRGenerator(crd orktypes.CRDEntry) *CRGenerator {
	return &CRGenerator{crd: crd}
}

// Generate produces an example CR as a plain map (ready for YAML encoding).
// Required fields are filled with typed placeholders.
// Optional fields show their default values.
func (g *CRGenerator) Generate() map[string]interface{} {
	spec := g.buildSpec()

	return map[string]interface{}{
		"apiVersion": g.crd.APITypes.Group + "/" + g.crd.APITypes.Version,
		"kind":       g.crd.APITypes.Kind,
		"metadata": map[string]interface{}{
			"name":      strings.ToLower(g.crd.APITypes.Kind) + "-sample",
			"namespace": "default",
		},
		"spec": spec,
	}
}

func (g *CRGenerator) buildSpec() map[string]interface{} {
	spec := make(map[string]interface{})

	// Required fields — from validation deny+exists rules
	if v := g.crd.EffectiveValidation(); v != nil {
		for _, rule := range v.Rules {
			if rule.Action != "deny" || string(rule.Operator) != "exists" {
				continue
			}
			path := trimSpecPrefix(rule.Field)
			if path == "" || strings.Contains(path, ".") {
				continue
			}
			if _, exists := spec[path]; !exists {
				spec[path] = placeholderFor(path, "string")
			}
		}
	}

	// Optional fields — from mutation defaults
	if m := g.crd.EffectiveMutation(); m != nil {
		for _, rule := range m.Rules {
			path := trimSpecPrefix(rule.Field)
			if path == "" || strings.Contains(path, ".") {
				continue
			}
			if rule.Default != nil {
				if _, exists := spec[path]; !exists {
					spec[path] = rule.Default
				}
			}
		}
	}

	return spec
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

type fieldInfo struct {
	name        string
	typ         string
	required    bool
	description string
	defaultVal  interface{}
}

// specFieldRegexp matches {{ .spec.fieldName }} in template expressions.
// Handles: {{ .spec.image }}, {{.spec.replicas}}, {{ .spec.some.nested }}
var specFieldRegexp = regexp.MustCompile(`{{\s*\.spec\.([a-zA-Z][a-zA-Z0-9_]*)\s*(?:[^}]*)}}`)

func extractSpecPaths(templates []string) []string {
	seen := map[string]bool{}
	var paths []string
	for _, tmpl := range templates {
		for _, match := range specFieldRegexp.FindAllStringSubmatch(tmpl, -1) {
			if len(match) >= 2 && !seen[match[1]] {
				seen[match[1]] = true
				paths = append(paths, match[1])
			}
		}
	}
	sort.Strings(paths)
	return paths
}

func trimSpecPrefix(field string) string {
	field = strings.TrimPrefix(field, "spec.")
	if strings.HasPrefix(field, "spec") {
		return ""
	}
	return field
}

func inferTypeFromValue(v interface{}) string {
	switch v.(type) {
	case int, int32, int64, float32, float64:
		return "integer"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "string"
	}
}

func placeholderFor(name, typ string) interface{} {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "image"):
		return "my-image:latest"
	case strings.Contains(lower, "replica"):
		return 1
	case strings.Contains(lower, "port"):
		return 8080
	case strings.Contains(lower, "region"):
		return "us-east-1"
	case strings.Contains(lower, "domain"):
		return "example.com"
	case strings.Contains(lower, "url") || strings.Contains(lower, "uri"):
		return "https://example.com"
	case strings.Contains(lower, "step"):
		return []interface{}{map[string]interface{}{"name": "step-1", "command": "echo hello"}}
	case typ == "integer":
		return 1
	case typ == "boolean":
		return false
	case typ == "array":
		return []interface{}{}
	default:
		return "my-" + lower
	}
}

func toTitle(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func strPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32 { return &i }
func boolPtr(b bool) *bool    { return &b }

func toRawExtension(v interface{}) (*apiextv1.JSON, error) {
	// Simple conversion for primitive types
	switch typed := v.(type) {
	case string:
		return &apiextv1.JSON{Raw: []byte(`"` + typed + `"`)}, nil
	case int, int64:
		return &apiextv1.JSON{Raw: []byte(fmt.Sprintf("%d", typed))}, nil
	case float64:
		return &apiextv1.JSON{Raw: []byte(fmt.Sprintf("%g", typed))}, nil
	case bool:
		if typed {
			return &apiextv1.JSON{Raw: []byte("true")}, nil
		}
		return &apiextv1.JSON{Raw: []byte("false")}, nil
	}
	return nil, nil
}
