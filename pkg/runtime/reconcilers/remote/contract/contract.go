// Package contract defines the transport-agnostic types shared across all
// remote reconciler implementations (http, grpc, …).
package contract

// Result is the response a remote reconciler returns after each reconcile cycle.
// HTTP reconcilers decode it from JSON; gRPC reconcilers will map from a protobuf message.
type Result struct {
	// Result is one of "ok", "requeue", or "error".
	Result string `json:"result"`
	// RequeueAfter is the requeue duration string (e.g. "60s"). Required when result is "requeue".
	RequeueAfter string `json:"requeueAfter,omitempty"`
	// Error is a human-readable error message. Non-empty triggers backoff retry.
	Error string `json:"error,omitempty"`
	// Status is an optional map of fields to patch onto the CR's status.
	// Merged with any katalog emit.status patch; remote fields win on conflict.
	Status map[string]interface{} `json:"status,omitempty"`
	// Resources is an optional list of Kubernetes objects to apply via SSA.
	// Orkestra sets the CR as owner for same-namespace resources.
	Resources []map[string]interface{} `json:"resources,omitempty"`
}
