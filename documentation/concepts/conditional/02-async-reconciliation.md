# Async Reconciliation

Orkestra's reconciler is synchronous — it receives a CR key, runs, and returns. Real systems are not: a Deployment takes time to become ready, a database takes time to accept connections.

The `when:` gate on `onReconcile` bridges this gap. If the condition is not met, the block is skipped and the CR is automatically requeued when the child resource's state changes. No sleeping. No polling. No custom state machine.

---

## The pattern

```yaml
operatorBox:
  reconcile:
    onCreate:
      deployments:
        - name: "{{ .metadata.name }}"
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          reconcile: true
    onReconcile:
      services:
        - name: "{{ .metadata.name }}-svc"
          port: 80
          targetPort: 8080
          reconcile: true
          when:
            - field: children.deployment.status.readyReplicas
              equals: "{{ .spec.replicas }}"
  crdFile: my-operator-crd.yaml
  # Phase 1 — runs once on CR creation
  # Phase 2 — runs only when Phase 1 is complete
```

What happens:

1. `onCreate` runs — Deployment is created
2. `onReconcile` evaluates `when:` — Deployment not ready yet — block skipped, CR requeued
3. Deployment pods become Ready — watch event fires — CR requeued
4. `onReconcile` evaluates `when:` — condition passes — Service is created
5. Every subsequent reconcile: Deployment still ready — Service is re-applied (drift correction)

---

## How requeue works without explicit calls

**Watch events.** When a child resource's status changes — pods becoming Ready, a Job completing, a LoadBalancer receiving an IP — the informer fires an event that requeues the owning CR automatically. Orkestra sets owner references and `orkestra-owner` labels on every child resource so the watch link is always established.

**Resync.** Each operatorBox has a `resync:` interval (default 15s). At each tick, all known CRs are re-enqueued. This is the backstop — even if a watch event is delayed or missed, the CR reconciles within one resync cycle.

You never need `ctrl.Result{RequeueAfter: N}`. The runtime handles it.

---

## The `.children.*` namespace

The `when:` gate references child resource status via the `.children.*` resolver namespace:

| Expression | What it returns |
|---|---|
| `{{ readyReplicas .children.deployment }}` | Number of ready pods |
| `{{ desiredReplicas .children.deployment }}` | Total desired pods |
| `{{ jobSucceeded .children.job }}` | `true` when Job completed successfully |
| `{{ serviceLoadBalancerIP .children.service }}` | LoadBalancer ingress IP |
| `{{ serviceLoadBalancerHost .children.service }}` | LoadBalancer ingress hostname |
| `{{ get .children.cronjob "status" "lastScheduleTime" }}` | Last scheduled run |

The same `.children.*` data is available in `status.fields` expressions — giving the CR's status a live view of its children.

---

## Multiple ordered phases

```yaml
onReconcile:
  # Phase A — always runs
  configMaps:
    - name: "{{ .metadata.name }}-config"
      reconcile: true
  # Phase B — runs only when Deployment is ready
  services:
    - name: "{{ .metadata.name }}-svc"
      reconcile: true
      when:
        - field: children.deployment.status.readyReplicas
          equals: "{{ .spec.replicas }}"

  # Phase C — runs only when Service has a LoadBalancer IP
  ingresses:
    - name: "{{ .metadata.name }}-ingress"
      host: "{{ .spec.host }}"
      reconcile: true
      when:
        - field: children.service.status.loadBalancer
          operator: exists
```

The first phase has no gate — runs unconditionally. The second waits for the Deployment. The third waits for the Service's LoadBalancer IP.

---

## What this replaces

In a traditional Go operator, three-phase reconciliation requires: a phase annotation on CR status, a switch statement routing per-phase handlers, each handler returning `ctrl.Result{RequeueAfter: 5s}`, explicit existence checks, idempotency proofs, and tests for each phase transition.

In Orkestra: two resource groups and one `when:` condition.
