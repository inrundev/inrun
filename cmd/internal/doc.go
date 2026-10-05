// Package internal wires the runtime and gateway processes together. It
// builds every component, threads their dependencies, and hands them to
// pkg/process to start and stop in order.
//
// KonductRuntime starts the runtime (informers, reconcile loop, leader
// election). KonductGateway starts the in-cluster gateway with TLS and
// admission webhooks. KonductGatewayDev starts the gateway's HTTP API locally,
// without TLS or webhooks, for the developer build.
package internal
