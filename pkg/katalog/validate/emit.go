package validate

import (
	"fmt"
	"text/template"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateEmit checks operatorBox.emit.events across all enabled CRDs.
//
// Enforces:
//  1. type must be a valid EmitEventType (Normal or Warning).
//  2. reason must be non-empty.
//  3. message must be non-empty and a valid Go template when it contains "{{".
//  4. when/or condition field values are valid Go templates when they contain "{{".
func (e *executor) validateEmit() error {
	funcMap := buildFuncMapForValidation(e.k.Notes)
	for crdName, crd := range e.k.EnabledCRDs() {
		for name, entry := range crd.OperatorBox.EmitEntries() {
			if err := validateEmitEventEntry(crdName, name, entry, funcMap); err != nil {
				return err
			}
		}
		if err := validateStatusConfig(crdName, "operatorBox.status", crd.OperatorBox.Status); err != nil {
			return err
		}
		if crd.OperatorBox.Emit != nil {
			if err := validateStatusConfig(crdName, "operatorBox.emit.status", crd.OperatorBox.Emit.Status); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateEmitEventEntry(crdName, name string, entry *orktypes.EmitEventEntry, funcMap template.FuncMap) error {
	if entry == nil {
		return nil
	}
	if !orktypes.IsValidEmitEventType(entry.Type) {
		return fmt.Errorf("%s crd %q: emit.events[%q]: type %q is invalid — valid values: %s",
			failureMark(), crdName, name, entry.Type, orktypes.EmitEventTypesJoined())
	}
	if entry.Reason == "" {
		return fmt.Errorf("%s crd %q: emit.events[%q]: reason must not be empty",
			failureMark(), crdName, name)
	}
	if entry.Message == "" {
		return fmt.Errorf("%s crd %q: emit.events[%q]: message must not be empty",
			failureMark(), crdName, name)
	}
	if isTemplate(entry.Message) {
		if err := validateTemplate("emit.events", crdName, name, "message", entry.Message, funcMap); err != nil {
			return err
		}
	}
	for i, c := range entry.When {
		if isTemplate(c.Field) {
			if err := validateTemplate("emit.events", crdName, name, fmt.Sprintf("when[%d].field", i), c.Field, funcMap); err != nil {
				return err
			}
		}
	}
	for i, c := range entry.Or {
		if isTemplate(c.Field) {
			if err := validateTemplate("emit.events", crdName, name, fmt.Sprintf("or[%d].field", i), c.Field, funcMap); err != nil {
				return err
			}
		}
	}
	return nil
}
