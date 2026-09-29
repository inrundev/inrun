# Admission

The Gateway serves the admission webhook endpoints that Kubernetes calls on every resource create and update. Admission rules declared in a Katalog are enforced here — synchronously, before the object is stored in etcd.

---

## Validation webhooks

A `ValidatingWebhookConfiguration` is registered automatically when at least one `validation.rules` entry is declared in the Katalog. The Gateway evaluates each rule against the incoming object and returns allow or deny.

Rule types available:

| Type | Declaration | Behavior |
|------|-------------|----------|
| Prefix check | `prefix: "registry.internal/"` | Denies if field value does not start with the prefix |
| Numeric bound | `greaterThan: 0` | Denies if field value is not greater than the bound |
| Enum membership | `operator: in` | Denies if value is not in the declared set |
| Existence | `operator: exists` | Warns or denies if field is absent |
| Uniqueness | `operator: unique` | Denies if another CR of the same kind already holds the value |
| External | `external: <endpoint>` | Calls an external HTTP endpoint to make the decision |

Each rule carries an `action: deny` or `action: warn`. Deny rules block the apply. Warn rules allow it through and surface the message in the response.

Rules can be conditional:

```yaml
admission:
  validation:
    rules:
      - field: spec.productionApproval
        operator: exists
        message: "production deployments require an approval ticket"
        action: deny
        when:
          field: spec.environment
          value: production
```

The `when:` guard is evaluated first. If it does not match, the rule is skipped entirely.

---

## Mutation webhooks

A `MutatingWebhookConfiguration` is registered when at least one `mutation.rules` entry is declared. Mutation runs before validation — a field defaulted by a mutation rule is visible to validation rules in the same request.

```yaml
admission:
  mutation:
    mutateFirst: true
    rules:
      - field: spec.replicas
        default: 2
        valueType: int

      - field: spec.environment
        default: "development"
```

Rule types:

- **`default`** — sets the field only when it is absent or empty
- **`override`** — always sets the field, regardless of the current value

`valueType` preserves the native type in the patch: `int`, `bool`, `float`, or `string` (default). Use it for fields the CRD schema expects as non-string types.

---

## Deletion protection

Deletion protection prevents a CR from being deleted while the operator is running and while finalizers are present. Configured at the Katalog level:

```yaml
security:
  admission:
    webhooks:
      deletion:
        enabled: true
```

The Gateway registers a `ValidatingWebhookConfiguration` that intercepts DELETE requests and denies them when the CR has Orkestra-managed finalizers. This is separate from the admission validation webhook.

---

## Failure policy

Each webhook endpoint can be configured independently:

```yaml
security:
  admission:
    webhooks:
      admission:
        enabled: true
      failurePolicy: Ignore
```

`Ignore` — if the Gateway is unreachable, the request is allowed through. Admission degrades gracefully; the Runtime re-validates at reconcile time.

`Fail` — if the Gateway is unreachable, the request is blocked. Use this for deletion protection where liveness of the webhook is a security property.

---

## Certificate management

The Gateway generates and rotates its own TLS certificates. No cert-manager is required.

At startup the Gateway reads or creates a certificate in a configured Kubernetes Secret. It patches the `caBundle` field of every registered webhook configuration to match. On rotation — triggered by configurable threshold and validity period — it repeats the process without requiring a restart or redeployment.

---

## RBAC

`ork generate bundle --for gateway` produces a ClusterRole scoped to:

- Reading and patching `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration`
- Reading and writing the certificate Secret
- Reading Katalog-declared CRDs (to evaluate admission rules)

It receives no permissions for child resources (Deployments, Services, etc.) and no write access to CRDs it does not own.
