package katalog

import (
	"github.com/orkspace/orkestra/pkg/logger"
)

// Debug katalog information from merger
func (k *Katalog) DebugKatalogInformation() {
	// [DEBUG] Contents of k.Security
	logger.Debug().Interface("katalog security", k.Security).Msg("katalog security")

	// [DEBUG] Contents of k.Notiication
	logger.Debug().Interface("katalog notification", k.Notification).Msg("katalog notification")

	// [DEBUG] Contents of k.Spec
	logger.Debug().Interface("katalog spec", k.Spec).Msg("katalog spec")

	// [DEBUG] Contents of k.enabledCRDs
	logger.Debug().Interface("katalog enabledCRDs", k.enabledCRDs).Msg("katalog enabledCRDs")

	// [DEBUG] Contents of k.metadata
	logger.Debug().Interface("katalog metadata", k.metadata).Msg("katalog metadata")
}
