# Policy

Platform teams need to control which patterns developers can import and which container images operators can deploy. Orkestra handles both through the platform-admission Motif and admission webhooks.

---

## The platform-admission Motif

The platform-admission Motif contributes validation and mutation rules to every CRD in a Komposer. Instead of each Katalog independently enforcing policy, a shared Motif carries the rules — versioned and distributed alongside every other pattern.

```yaml
# platform-admission motif: enforces image registry policy
inputs:
  - name: allowedRegistry
    default: "ghcr.io/myorg/"
    description: Only images from this registry prefix are allowed

resources:
  admission:
    validation:
      - field: spec.image
        operator: startsWith
        value: "{{ .inputs.allowedRegistry }}"
        message: "spec.image must be from {{ .inputs.allowedRegistry }}"
        action: deny
```

Any Katalog that imports this Motif inherits the image policy. The allowed registry is an input — platform teams set it per-environment.

---

## Import in a Katalog

```yaml
# katalog.yaml — webapp-operator with platform policy
spec:
  crds:
    webapp:
      imports:
        - motif: oci://ghcr.io/myorg/motifs/web-service:v1.0.0
          with:
            image: "{{ .spec.image }}"
            port: "{{ .spec.port }}"

        - motif: oci://ghcr.io/myorg/motifs/platform-admission:v1.0.0
          with:
            allowedRegistry: "ghcr.io/myorg/"
```

Now any CR with `spec.image` outside `ghcr.io/myorg/` is denied at admission time — before the reconciler runs.

---

## Admission webhooks

Enable Orkestra's admission webhook in the Komposer:

```yaml
# komposer.yaml
security:
  admission:
    webhooks:
      admission:
        enabled: true
gateway:
  endpoint: http://orkestra-gateway.orkestra-system.svc:8080
```

With the webhook enabled, every CR apply hits Orkestra's `/validate` and `/mutate` endpoints before etcd. Policy violations return a user-facing error immediately — no CR is created, no reconcile runs.

Deploy with Gateway:

```bash
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --set gateway.enabled=true \
  --wait --timeout 120s
```

---

## Verify the policy

```bash
# This CR uses an image from an allowed registry — accepted
kubectl apply -f cr-valid.yaml
# webapp.rkguide.demo/my-webapp created

# This CR uses nginx — rejected at admission
kubectl apply -f cr-denied.yaml
# Error: admission webhook denied:
#   spec.image must be from ghcr.io/myorg/
```

The policy enforcement happens at the webhook layer — no CR hits the reconciler, no Deployment is attempted.

---

## The supply chain trust model

`ork inspect` tells you what gates a pattern passed before it was published. Platform teams use this to enforce a quality floor before patterns can be imported into production Komposers:

1. **Inspect before importing.** Check `Simulate` and `E2E` status. A pattern with `⊘ Skipped` on both gates has made no verifiable claims.

2. **Set `ORK_REGISTRY` to your internal registry.** Publish patterns from the community registry only after reviewing them. Your `ORK_REGISTRY` is the boundary between "trusted" and "unreviewed."

3. **Use platform-admission for image policy.** Operators deployed from your registry should only be able to run images from your approved registries.

4. **`ork plan` before upgrading.** See exactly what resource changes an upgrade introduces before applying it. Non-destructive changes are safe. Destructive ones — removed fields, changed CRD groups — require coordination.

---

## Bad-actor patterns

A well-crafted pattern can pass simulate assertions while doing something unexpected. `simulate.yaml` only asserts what its `expect:` block declares — if the author omits a ClusterRoleBinding from the assertions, simulate passes. The resource is still created.

`ork plan` shows every resource an operator would create — including ones simulate didn't assert:

```bash
ork plan -f katalog.yaml
# + deployments/my-webapp                  ← expected
# + clusterrolebindings/my-webapp-admin    ← not in simulate assertions
```

Review `ork plan` output for patterns from external sources before importing them into production. Simulate passing does not mean the operator is well-behaved — it means the simulate assertions passed.

---

## Try it

```bash
ork init --pack registry-guide
cd 04-katalog-platform

# Follow the steps in the README
```

→ Next: [Lifecycle](10-lifecycle.md)
