package config

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// -----------------------------------------------------------------------------
func Validate() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate
}

// Normalize environment
func (k *Config) normalizeEnvironment() {
	// Normalize inrun environment
	switch strings.ToLower(k.inrun.environment) {
	case DevShort, Development:
		k.inrun.environment = Development
	case StagingShort, Staging:
		k.inrun.environment = Staging
	case Live, ProdShort, Production:
		k.inrun.environment = Production
	default:
		k.inrun.environment = Development
	}
}
