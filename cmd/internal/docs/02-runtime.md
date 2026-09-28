# 02 — Runtime

The runtime is the full Orkestra reconcile loop. It is assembled in
`konstructRuntime` in `runtime_konstructor.go` and started by `KonductRuntime` in `runtime.go`.

## Wiring diagram

```
Katalog (YAML)
    │
    ▼
merger → katalog.Katalog
    │
    ▼
kubeclient.Kubeclient          REST config, dynamic client, typed clientset.
    │                          Started immediately — everything else needs it.
    │
    ├──► ClientProvider         One REST client constructor per CRD. Deferred.
    │
    ├──► SharedInformerFactory  One SharedIndexInformer per CRD. Populates
    │        │                  in-memory cache from watch stream on Start().
    │        └──► per-CRD informer
    │
    ├──► KordinatorRegistry     Maps GVK → (CRD, informer, reconcilerFactory).
    │        └──► per-CRD reconciler factory closure
    │
    ├──► DependencyKordinator   Starts workers in topological order.
    │
    └──► HealthServer           HTTP server — /ready, /livez, /katalog routes.
```

`konstructRuntime` is intentionally one long function. The full dependency graph
is visible in one place; splitting it across files would make start order harder
to trace.

## Komponent start order

Orkestra calls `Start()` on each komponent in registration order and `Stop()` in
reverse on shutdown. The runtime registers:

1. `HealthServer` — serves `/ready` and `/livez` immediately, before any other
   komponent is ready. External load balancers can probe this from the first
   second of startup.
2. `Kubeclient` — already started during wiring, but registered so Orkestra
   manages its `Stop()`.
3. `Event` — the Kubernetes event recorder. Requires a live kubeclient.
4. `QueueRegistry` — per-CRD bounded work queues.
5. `DefaultWorkqueue` — the shared unbounded queue for CRDs that opt into
   `shared.queue: true`.
6. `SharedInformerFactory` — opens watches against the API server and populates
   in-memory caches. Closes the ready channel when all informers are synced.
7. `DependencyKordinator` — waits for informer sync, then starts CRD workers in
   topological dependency order.

## KonductRuntime and leader election

`KonductRuntime` wraps `konstructRuntime` with a `konductor.NewKonductorElection`.
Leader election uses a Kubernetes Lease object. Only the replica holding the
lease calls `kord.Kordinate(ctx)`, which starts the CRD workers. All other
replicas start the health server and informers but do not start workers.

Leader election is necessary because reconcilers modify cluster state. Running
two replicas concurrently would cause conflicting writes and unpredictable
resource status. The gateway does not have this problem — webhook admission
decisions are idempotent.

→ Back: [01-overview.md](01-overview.md)
→ Next: [03-gateway.md](03-gateway.md)
