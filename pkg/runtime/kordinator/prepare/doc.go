// Package prepare builds the domain.PreparedRequest for one reconcile cycle:
// namespace guard, defaults, mutations and validation. The kordinator and
// simulate both call Prepare, so simulations run the same logic as a cluster.
package prepare
