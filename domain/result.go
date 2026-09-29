package domain

import "time"

// Result is returned by Reconcile to signal post-reconcile scheduling intent.
// Zero value means: no special scheduling — the runtime's resync and queue handle it.
type Result struct {
	// RequeueAfter schedules an exact re-enqueue of this object after the given duration.
	// Ignored if zero. Honored only on successful reconcile; error path uses retryBackoff.
	RequeueAfter time.Duration

	// StatusPatch is an optional set of fields to merge into the CR's status after
	// reconcile. Keys are applied after katalog emit.status fields — remote fields win
	// on conflict. Ignored when nil or empty.
	StatusPatch map[string]interface{}
}
