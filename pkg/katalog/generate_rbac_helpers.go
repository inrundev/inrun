package katalog

import (
	"github.com/orkspace/orkestra/pkg/children"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// HasConversionPaths returns true only if conversion is enabled and at least
// one CRD declares conversion paths.
func (k *Katalog) HasConversionPaths() bool {
	if k == nil || k.konfig == nil {
		logger.Debug().Msg("katalog or konfig is nil")
		return false
	}
	if !k.IsConversionEnabled() {
		logger.Debug().Msg("conversion is disabled")
		return false
	}
	for _, crd := range k.Enabled() {
		if crd.EffectiveConversion() == nil {
			logger.Debug().Str("crd", crd.Name).Msg("conversion is nil")
			continue
		}
		if len(crd.EffectiveConversion().Paths) > 0 {
			logger.Debug().Str("crd", crd.Name).Msg("conversion has paths")
			return true
		}
	}
	return false
}

// HasValidationRules returns true if admission is enabled and at least one CRD
// declares validation rules.
func (k *Katalog) HasValidationRules() bool {
	if k == nil {
		return false
	}
	if !k.IsAdmissionEnabled() {
		logger.Debug().Msg("admission is disabled")
		return false
	}
	for _, crd := range k.Enabled() {
		if crd.EffectiveValidation() == nil {
			continue
		}
		if len(crd.EffectiveValidation().Rules) > 0 {
			logger.Debug().Str("crd", crd.Name).Msg("validation has rules")
			return true
		}
	}
	return false
}

// HasMutationRules returns true if admission is enabled and at least one CRD
// declares mutation rules.
func (k *Katalog) HasMutationRules() bool {
	if k == nil {
		return false
	}
	if !k.IsAdmissionEnabled() {
		logger.Debug().Msg("admission is disabled")
		return false
	}
	for _, crd := range k.Enabled() {
		if crd.EffectiveMutation() == nil {
			continue
		}
		if len(crd.EffectiveMutation().Rules) > 0 {
			logger.Debug().Str("crd", crd.Name).Msg("mutation has rules")
			return true
		}
	}
	return false
}

// HasValidationOrMutationRules returns true if any CRD has validation or mutation rules.
func (k *Katalog) HasValidationOrMutationRules() bool {
	if k == nil {
		return false
	}
	return k.HasValidationRules() || k.HasMutationRules()
}

// Uses reports whether any enabled CRD uses the given resource in a hook
// template block. Accepts both singular ("deployment") and plural ("deployments")
// keys, as well as shorthands ("hpa"). Drives off builtInRegistry — no separate
// table to maintain.
func (k *Katalog) Uses(resource string) bool {
	b, ok := children.LookupBuiltInByResource(resource)
	if !ok {
		return false
	}
	return k.anyDetects(b.Detect)
}

// anyDetects returns true if detect returns true for any enabled CRD.
func (k *Katalog) anyDetects(detect func(orktypes.CRDEntry) bool) bool {
	for _, crd := range k.Enabled() {
		if detect(crd) {
			return true
		}
	}
	return false
}
