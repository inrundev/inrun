# Composing Patterns

A Komposer is how patterns from the registry become a running platform. It imports multiple Katalogs, applies environment-specific overrides, and generates the runtime bundle that Orkestra executes.

---

## The import block

```yaml
# komposer.yaml
imports:
  registry:
    - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0
    - oci://ghcr.io/myorg/katalogs/cache-operator:v1.0.0
    - oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
```

Every registry entry resolves the `katalog.yaml` from the OCI artifact. The Komposer merges all CRD entries into one runtime. One Orkestra binary. Multiple operators.

To use the upstream `komposer.yaml` instead (resolves its own imports recursively):

```yaml
imports:
  registry:
    - url: oci://ghcr.io/myorg/patterns/postgres:v1.0.0
      useKomposer: true
```

---

## Override without touching upstream

Your inline `spec.crds` block always wins over imported values:

```yaml
imports:
  registry:
    - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0

spec:
  crds:
    webapp:
      operatorBox:
        reconcile:
          workers: 8         # production override
          resync: 30s
```

This applies to any CRD-level field: `apiTypes`, `operatorBox.reconcile.workers`, `operatorBox.reconcile.resync`, `webhooks`, `deletionProtection`. The upstream operator is unmodified. Your override is applied at runtime.

---

## The supply chain: twenty lines for a full platform

```yaml
# 06-pattern-zoo/komposer.yaml
imports:
  registry:
    - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0
    - oci://ghcr.io/myorg/katalogs/cache-operator:v1.0.0
    - oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
    - oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/kafka:v3.6.0
    - oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/redis:v7.2.0
```

Five operators. All versioned. All with quality signals you inspected before importing. Before Orkestra, each of these was a separate controller image, Helm chart, and operational burden. Here they compose into one runtime that manages five CRDs.

---

## Deploy

Generate the runtime bundle (ConfigMap + RBAC) and apply it:

```bash
ork generate bundle -f komposer.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

Install Orkestra (or upgrade to pick up the new bundle):

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --wait --timeout 120s
```

---

## Orkestra does not auto-sync

Assuming you update your `komposer.yaml`, you regenerate and apply the bundle. After `kubectl apply -f bundle.yaml`, Orkestra is not yet running the new composition. The runtime reads the ConfigMap at startup. A rollout restart is the explicit signal: "I am ready to serve the new contract."

```bash
kubectl rollout restart deployment/orkestra-runtime -n orkestra-system
kubectl rollout status deployment/orkestra-runtime -n orkestra-system
```

This is intentional. The explicit restart makes upgrades auditable and reversible. See: [Orkestra Doesn't Auto-Sync — By Design](https://orkestra.sh/docs/foundations/no-autosync)

---

## Rollback

Revert `komposer.yaml` to the previous import versions, regenerate, apply, restart:

```bash
# revert komposer.yaml
ork generate bundle -f komposer.yaml -o bundle-rollback.yaml
kubectl apply -f bundle-rollback.yaml
kubectl rollout restart deployment/orkestra-runtime -n orkestra-system
```

The previous artifact versions are still in the registry — immutable, untouched. Rollback is a `git revert` and three commands.

---

## Try it

```bash
ork init --pack registry-guide

# Single-operator composition
cd 05-komposer
cat komposer.yaml
ork generate bundle -f komposer.yaml -o bundle.yaml
kubectl apply -f bundle.yaml

# Multi-pattern composition
cd ../06-pattern-zoo
cat komposer.yaml
ork generate bundle -f komposer.yaml -o bundle.yaml
```

→ Next: [Upgrading Patterns](07-upgrade.md)
