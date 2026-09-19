# Admission webhooks

Orkestra's Gateway serves as the admission layer for all CRs managed by your Katalog. Before any CR is written to the cluster, the Kubernetes API server calls the Gateway to decide whether to allow it, modify it, or reject it.

This covers three capabilities powered by the Gateway:

- **Validation** — reject CRs that violate your rules
- **Mutation** — fill in defaults before validation runs
- **Conversion** — translate CRs between schema versions (CRD versioning)

---

## How it connects

When you include `security.webhooks.admission.enabled: true` in your Katalog, the runtime or gateway registers webhook configurations with the API server:

```yaml
security:
  admission:
    webhooks:
      admission:
        enabled: true
      failurePolicy: Ignore
```

The Gateway receives every `CREATE` and `UPDATE` request for the CRs you declared, makes its decision, and returns a response. The API server stores (or rejects) the object based on that response.

Orkestra creates a `ValidatingWebhookConfiguration` only when you declare at least one validation rule. A `MutatingWebhookConfiguration` only when you declare at least one mutation rule. No rules → no webhook registered, no API server overhead.

---

## Validation

Validation rules gate CRs at admission time. A `deny` rule rejects the request before the object is stored in etcd. A `warn` rule logs the violation but lets the request through.

```yaml
crds:
  platform:
    admission:
      validation:
        rules:
          - field: spec.image
            prefix: "registry.internal/"
            message: "images must come from the internal registry"
            action: deny

          - field: spec.replicas
            greaterThan: 0
            message: "replicas must be at least 1"
            action: deny

          - field: spec.rateLimit
            operator: exists
            message: "declare spec.rateLimit for production readiness"
            action: warn
```

When a user applies a CR with `spec.image: docker.io/nginx:latest`, the webhook returns:

```text
Error from server: admission webhook "admission.orkestra.orkspace.io" denied the request:
[platform/my-platform] spec.image: images must come from the internal registry (registry.internal/)
```

The object is never written to etcd. The reconciler never sees it.

### Validation at reconcile time too

The reconciler re-runs all `deny` rules on every reconcile cycle. If a CR somehow bypassed the webhook (for example, during the brief window when the webhook was restarting), the reconciler will not create any child resources. No Deployment, no ConfigMap, no Service. The CR exists in the API server but Orkestra treats it as invalid.

---

## Mutation

Mutation rules fill in defaults before the CR is validated. A CR that omits `spec.replicas` receives the default value `2` from the mutating webhook, and then passes the `greaterThan: 0` validation rule without error.

```yaml
crds:
  platform:
    admission:
      mutation:
        mutateFirst: true
        rules:
          - field: spec.replicas
            default: 2
            valueType: int

          - field: spec.environment
            default: "development"

          - field: spec.rateLimit
            default: 100
            valueType: int
```

`mutateFirst: true` ensures defaults are applied before any `deny` rule fires, both at admission time and at reconcile time.

The mutated values appear in the CR's status so you can always see what Orkestra filled in:

```bash
kubectl get platform my-platform -o jsonpath='{.status}'
# {"environment":"development","phase":"Running","rateLimit":100,"replicas":2}
```

### Example: what the API server receives vs what the user sent

User applies:
```yaml
spec:
  image: registry.internal/my-app:v1.0.0
```

After mutation:
```yaml
spec:
  image: registry.internal/my-app:v1.0.0
  replicas: 2          # filled in by mutation webhook
  environment: development  # filled in by mutation webhook
  rateLimit: 100       # filled in by mutation webhook
```

---

## Conversion

Conversion webhooks allow CRDs to have multiple API versions simultaneously (`v1alpha1`, `v1beta1`, `v1`). The Gateway translates CRs between versions transparently as you evolve your schema.

```yaml
crds:
  platform:
    admission:
      conversion:
        strategy: Webhook
```

When a client requests a CR at `v1` and the object was stored at `v1alpha1`, the API server calls the Gateway to convert between the two. Clients always see the version they asked for, regardless of what version the object was originally created at.

---

## failurePolicy

```yaml
security:
  admission:
    webhooks:
      failurePolicy: Ignore   # or: Fail
```

- **`Ignore`** — if the Gateway is temporarily unreachable (e.g., during a rolling restart), the CR is accepted. Use this when availability takes priority.
- **`Fail`** — if the Gateway is unreachable, the CR is rejected. Use this in high-security environments where no CR should reach etcd without inspection.

Combined with `pdb.minAvailable: 1` in the Helm chart, the Gateway maintains at least one pod available during rolling restarts, making the window where `Fail` would block requests very narrow.

---

## Webhook self-healing

Orkestra's housekeeper watches its own `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects. If either is deleted — accidentally or by an attacker — Orkestra detects the change immediately via a Kubernetes Watch and recreates it:

```bash
kubectl delete validatingwebhookconfiguration orkestra-admission
# Orkestra detects this and recreates it within one API server round-trip
```

A safety poll (default 30 seconds) runs in parallel as a backstop for drift the Watch misses.

---

## Working example

For a complete Katalog with validation, mutation, and an E2E test, run:

```bash
ork init --pack security
cd admission
ork e2e
```

Follow the steps in the README
---
