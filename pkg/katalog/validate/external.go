package validate

import (
	orkexternal "github.com/orkspace/orkestra/pkg/external"
)

func (e *executor) validateExternalCalls() error {
	for crdName, entry := range e.k.EnabledCRDs() {
		if ht := entry.OperatorBox.EffectiveOnReconcile(); ht != nil {
			if err := orkexternal.ValidateCalls(crdName, "onReconcile.external", ht.External); err != nil {
				return err
			}
		}
		if ht := entry.OperatorBox.EffectiveOnCreate(); ht != nil {
			if err := orkexternal.ValidateCalls(crdName, "onCreate.external", ht.External); err != nil {
				return err
			}
		}
		if r := entry.OperatorBox.Reconcile; r != nil && r.Hooks != nil {
			if err := orkexternal.ValidateCalls(crdName, "hooks.external", r.Hooks.External); err != nil {
				return err
			}
		}
		if v := entry.EffectiveValidation(); v != nil {
			if err := orkexternal.ValidateCalls(crdName, "validation.external", v.External); err != nil {
				return err
			}
		}
		if m := entry.EffectiveMutation(); m != nil {
			if err := orkexternal.ValidateCalls(crdName, "mutation.external", m.External); err != nil {
				return err
			}
		}
	}
	return nil
}
