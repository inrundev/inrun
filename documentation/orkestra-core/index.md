# Orkestra Core

Orkestra Core is composed of three independent components. Each has a single responsibility, a minimal footprint, and no shared state with the others. They discover each other only when explicitly told where to look.

---

## The three components

### [Runtime](./runtime/index.md)

The reconciliation engine. It watches Kubernetes resources, runs your operator logic, manages resource lifecycle, and exposes the `/katalog` API. Every operator built with Orkestra runs inside a runtime.

### [Gateway](./gateway/index.md)

The external-facing process. It serves admission and conversion webhooks, handles notifications, and owns all outbound I/O. It is stateless, runs as multiple replicas, and has no reconciliation logic — no reconcilers, no informers, no cluster state.

### [Control Center](./controlcenter/_index.md)

The observability layer. It connects to one or more runtimes, reads their `/katalog` APIs, and renders a live view of every operator, CRD, worker, and CR in your fleet.

---

## How they compose

The three components share nothing at rest. No shared database, no shared cache, no direct in-process calls.

- The **Control Center** does not know where the Runtime is until you tell it. Point it at a runtime URL and it reads everything it needs from the `/katalog` API.

- The **Gateway** does not know where the Runtime is either. When the Runtime needs to send a notification, it calls the Gateway's `/notify` endpoint. The Gateway owns dispatch — SMTP, Slack — and the Runtime never touches it again.

- The **Runtime** advertises the Gateway's location. Its `/katalog` response includes a `gatewayEndpoint` field. The Control Center follows that link to fetch webhook stats and merge them into the per-CRD view — without either component needing to be co-located.

This means you can run them on the same pod or on entirely separate deployments. The architecture does not care.


---

## Where to go next

- **[Runtime](./runtime/index.md)** — reconciliation engine, worker configuration, health model
- **[Gateway](./gateway/index.md)** — webhooks, notifications, TLS setup
- **[Control Center](./controlcenter/index.md)** — live operator dashboard
- **[Deploying](../deploying.md)** — running the full stack in a real cluster
