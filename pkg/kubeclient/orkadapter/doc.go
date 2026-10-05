// Package orkadapter is the boundary between controller-runtime and the
// Orkestra runtime. ToClient exposes a kubeclient as a controller-runtime
// client.Client, so reconcilers written for controller-runtime run unchanged
// while Orkestra owns informers, queues, workers and scheduling.
package orkadapter
