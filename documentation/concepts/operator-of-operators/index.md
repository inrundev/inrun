# Operator of Operators

The `custom:` block in an operatorBox creates Custom Resources — not Kubernetes primitives. This means one Orkestra operator can instantiate resources that are themselves managed by other Orkestra operators in the same Katalog.

This pattern is called **operator composition**: a single Orkestra binary can run many operators simultaneously, and any operator can spin up instances of the others as side-effects of its own reconcile logic.

---

## How it works

You define a parent CRD and a child CRD in the same Katalog. The parent's `onCreate.custom` block declares child CRs to create. Orkestra resolves template expressions from the parent CR and creates the child CR. The child operator picks it up and reconciles its own resources.

```yaml
spec:
  crds:
    workspace:
      crdFile: crd-workspace.yaml
      operatorBox:
        reconcile:
          onCreate:
            custom:
              - apiVersion: platform.example.io/v1alpha1
                kind: SecretVault
                metadata:
                  name: "{{ .metadata.name }}-vault"
                  namespace: "{{ .metadata.namespace }}"
                  namespaced: true
                spec:
                  workspaceName: "{{ .metadata.name }}"
                  encryption: "{{ .spec.encryption }}"
                hasStatus: false
    secretvault:
      crdFile: crd-secretvault.yaml
      operatorBox:
        reconcile:
          onCreate:
            deployments:
              - name: "{{ .metadata.name }}-api"
                image: secrets-api:latest
```

Apply a `Workspace` CR → Orkestra creates the `SecretVault` → the SecretVault operator creates the Deployment. Delete the `Workspace` → everything cascades away via owner references.

---

## Owner references (automatic)

Orkestra sets an owner reference on every child CR pointing back to the parent. You get cascade deletion for free:

```bash
kubectl delete workspace dev-team
# SecretVault dev-team-vault — garbage-collected automatically
# Deployment created by SecretVault operator — also gone
```

No `onDelete` hooks needed for the common case.

---

## `hasStatus`

Controls whether Orkestra reads child CR status back into the parent's template resolver:

| Value | Behaviour |
|---|---|
| `false` | Skip status read — saves an API call |
| `true` | Read child status — available as `.children.custom["<name>"].status` |
| omitted | Auto-detect via REST mapping |

Reference child status in parent templates:

```yaml
status:
  fields:
    - path: vaultPhase
      value: '{{ (index .children.custom (printf "%s-vault" .metadata.name)).status.phase }}'
```

---

## Conditional children

Gate child CR creation on parent spec values. The child is skipped when the condition fails and created when it passes on the next reconcile:

```yaml
custom:
  - apiVersion: platform.example.io/v1alpha1
    kind: CacheCluster
    metadata:
      name: "{{ .metadata.name }}-cache"
    spec:
      size: "{{ .spec.cache.size }}"
    when:
      - field: spec.cache.enabled
        equals: "true"
```

---

## Drift correction

By default, child CRs are created once. With `reconcile: true`, Orkestra re-applies the child spec on every parent reconcile — any drift is corrected within the resync window:

```yaml
custom:
  - apiVersion: example.io/v1alpha1
    kind: BackupPolicy
    reconcile: true
    metadata:
      name: "{{ .metadata.name }}-backup"
    spec:
      schedule: "{{ .spec.backup.schedule }}"
```

---

## `forEach:` fan-out

Create one child CR per element in a parent list field:

```yaml
custom:
  - apiVersion: storage.example.io/v1alpha1
    kind: Shard
    metadata:
      name: "{{ .metadata.name }}-{{ .shard.name }}"
    spec:
      shardName: "{{ .shard.name }}"
      region: "{{ .shard.region }}"
    forEach:
      field: spec.shards
      as: shard
```

This creates one `Shard` CR per entry in `spec.shards`. Each shard CR is managed by its own operator.

---

## Missing CRDs

If a target CRD is not yet installed when Orkestra starts, it logs a warning and skips that child gracefully. When the CRD appears, Orkestra refreshes its REST mapper automatically — no restart required.

---

## Try it

```bash
ork init --pack advanced
cd 16-custom-resources/01-single-child
```

Follow the README — it walks through a `Workspace` → `SecretVault` → `Deployment` composition with seven progressively more complex examples.
