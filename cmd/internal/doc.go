// Package internal wires the runtime and gateway processes together. It
// builds every component, threads their dependencies, and hands them to
// pkg/process to start and stop in order.
//
// RunRuntime starts the runtime (informers, reconcile loop, leader
// election). RunGateway starts the in-cluster gateway with TLS and
// admission webhooks. RunGatewayDev starts the gateway's HTTP API locally,
// without TLS or webhooks, for the developer build.
package internal
