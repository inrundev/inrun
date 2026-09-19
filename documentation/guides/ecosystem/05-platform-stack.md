# Platform Stack

## All Four in One, With Policy

The four operators — ArgoCD, cert-manager, Prometheus, Crossplane — each run independently in `00`–`03`. `04-platform-stack` composes them into a single runtime using a Komposer, and adds the gateway so admission enforcement and deletion protection apply across all four CRDs from one declaration.

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/04-platform-stack
```

---

## What you will learn

- How a Komposer imports multiple operators from a registry and runs them in one process
- What per-CRD isolation means when one operator crashes
- How gateway admission turns reconcile-time validation into synchronous rejection at `kubectl apply`
- How deletion protection blocks `kubectl delete` — and how to cleanly disable it
- How `ork e2e` tests the full chain against real ecosystem tool installations

---

## The Komposer

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform-stack
imports:
  registry:
    - oci://ghcr.io/myorg/katalogs/app-operator:v0.1.0
    - oci://ghcr.io/myorg/katalogs/security-operator:v0.1.0
    - oci://ghcr.io/myorg/katalogs/monitoring-operator:v0.1.0
    - oci://ghcr.io/myorg/katalogs/infra-operator:v0.1.0

gateway:
  endpoint: http://orkestra-gateway.orkestra-system.svc:8080

security:
  admission:
    webhooks:
      admission:
        enabled: true
  operatorBox:
    runtime:
      deletionProtection:
        enabled: true
```

Each operator keeps its own informer, workqueue, and worker pool. One CRD failing does not affect others. The gateway registers admission webhooks for all four CRDs — a single policy enforcement point across the full stack.

---

## Two enforcement points, side by side

| Without gateway (`ork run`) | With gateway (Helm, `gateway.enabled=true`) |
|---|---|
| Bad CR is stored in etcd | Bad CR is rejected at `kubectl apply` |
| Reconciler halts, writes `ValidationFailed` condition | API server returns error, CR never reaches etcd |
| Team sees the condition in Control Center | Team sees the rejection message immediately |

Both paths run the same validation rules from the Katalog. The gateway moves enforcement upstream.

---

## Admission in practice

Apply an `App` without team ownership — the Katalog requires `spec.labels.team`:

```bash
kubectl apply -f cr-denied.yaml
```

```text
Error from server: admission webhook "validate.orkestra.orkspace.io" denied the request:
validation denied: All apps must declare team ownership (spec.labels.team)
```

The CR is never created. ArgoCD never sees it.

---

## Deletion protection in practice

The Infra CRD is protected. Try to delete it:

```bash
kubectl delete infra protected-db
```

```text
Error from server: admission webhook "protect.resources.orkestra.orkspace.io" denied the request:
[Orkestra Security] The resource is protected from deletion.
```

To proceed, disable protection on the individual CR:

```bash
kubectl patch infra protected-db --type=merge \
  -p '{"metadata":{"annotations":{"orkestra.sh/deletion-protection":"false"}}}'
kubectl delete infra protected-db
```

To remove protection platform-wide, set `security.deletionProtection.enabled: false` in the Komposer, regenerate the bundle, and restart the gateway. The gateway's housekeeper removes the deletion-protection webhook automatically.

---

## E2E — testing the full abstraction chain

The ecosystem-composition pack includes an `e2e.yaml` that installs the real ecosystem tools in a kind cluster and then asserts the full chain works:

```yaml
# ecosystem-composition/e2e.yaml
imports:
  - ./00-argocd/e2e.yaml       # installs ArgoCD via Helm, applies App CR, asserts Application exists
  - ./01-cert-manager/e2e.yaml # installs cert-manager, applies SecurityConfig, asserts Certificate + Secret
  - ./02-prometheus/e2e.yaml   # installs kube-prometheus-stack, asserts ServiceMonitor + PrometheusRule
  - ./03-crossplane/e2e.yaml   # installs Crossplane, applies Infra CR, asserts PostgreSQLInstance
```

Each `e2e.yaml` uses `setup.helm` to install the ecosystem tool, then asserts:
1. The internal CRD (`App`, `SecurityConfig`, etc.) was created
2. The downstream ecosystem resource (ArgoCD `Application`, cert-manager `Certificate`, etc.) was created

This is not mocked. The e2e runs against real installations.

```bash
ork e2e -f e2e.yaml          # full suite
ork e2e -f 00-argocd/e2e.yaml  # single operator
```

---

## Try it

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/04-platform-stack
# Follow steps in README
```

→ [All-in-One](./06-all-in-one.md) — one `PlatformResource` CRD with a `workloadType` discriminator routing to all four tools.
