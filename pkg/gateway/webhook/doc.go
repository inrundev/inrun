// Package webhook serves the gateway's HTTPS admission and conversion
// webhooks: validation, mutation, conversion between CRD versions, and
// deletion and namespace protection. It registers and cleans up the webhook
// configurations in the cluster.
package webhook
