//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// validateCRDFile checks that path is a valid YAML file with the required
// CustomResourceDefinition fields: apiVersion, kind, spec.group, spec.names.kind.
func validateCRDFile(path string) error {
	data, err := readLocal(path)
	if err != nil {
		return err
	}
	var crd struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Spec       struct {
			Group string `yaml:"group"`
			Names struct {
				Kind string `yaml:"kind"`
			} `yaml:"names"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	if crd.Kind != "CustomResourceDefinition" {
		return fmt.Errorf("kind must be CustomResourceDefinition, got %q", crd.Kind)
	}
	if crd.Spec.Group == "" {
		return fmt.Errorf("spec.group is required")
	}
	if crd.Spec.Names.Kind == "" {
		return fmt.Errorf("spec.names.kind is required")
	}
	return nil
}
