// Package kordinator decides when each CRD's workers start, in dependency
// order, and runs every reconcile cycle: gate, prepare, maintain, reconcile,
// post. It also activates CRDs that appear after startup and restarts those
// that are deleted and recreated.
package kordinator
