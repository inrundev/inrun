# 16 — Custom Resources

Orchestrate third-party Custom Resources (CRs) declaratively from within your Orkestra operator. Your CRD acts as the parent; Orkestra creates, updates, and garbage-collects child CRs automatically, resolving Go-template expressions from the parent's fields.

This pattern is called **operator composition**: a single Orkestra binary can run many operators simultaneously, and any operator can spin up instances of the others as side-effects of its own reconcile logic.

---

## Examples

| # | Folder | What it teaches |
|---|--------|-----------------|
| 01 | [01-single-child](01-single-child/) | Create one child CR on parent creation. Owner references and cascade deletion. |
| 02 | [02-status-propagation](02-status-propagation/) | Read child CR status back into the parent's resolver context (`hasStatus: true`). |
| 03 | [03-conditional-children](03-conditional-children/) | `when:` conditions — create/skip/cleanup child CRs based on parent spec flags. |
| 04 | [04-drift-correction](04-drift-correction/) | `reconcile: true` — keep child CRs in sync across every reconcile, not just onCreate. |
| 05 | [05-forEach-sharding](05-forEach-sharding/) | `forEach:` fan-out — create N child CRs from a list field in the parent spec. |
| 06 | [06-full-platform-composition](06-full-platform-composition/) | Motif imports, three child CRs, and status aggregation into a full platform stack. |
| 07 | [07-multi-crd-pipeline](07-multi-crd-pipeline/) | Multiple child CRs + multiple controllers in one Katalog, driven by a Motif. |

Start with **01** and work forward. Each example is self-contained with its own CRDs, a sample CR, and a cleanup script.

---

## Quick Start

```bash
cd 01-single-child

# Apply the CRDs
kubectl apply -f crd-workspace.yaml
kubectl apply -f crd-secretvault.yaml

# Start the runtime
ork run

# use (--dev) to create a kind cluster and run orkestra

# In another terminal: apply a sample CR
kubectl apply -f cr.yaml

# Watch the cascade
kubectl get workspaces,secretvaults,deployments -n default
```

Each sub-folder's `README.md` explains the concepts introduced, shows the key YAML, and gives step-by-step instructions.

---

## Key Concepts

### `onCreate.custom` / `onReconcile.custom`

Declares child CRs to create when a parent CR is reconciled. Each entry is a full CR manifest with Go-template expressions resolved from the parent:

```yaml
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
          hasStatus: false
```

### `hasStatus`

Controls whether Orkestra reads child status back into the parent's template resolver context:

| Value | Behaviour |
|-------|-----------|
| `false` | Skip status read — saves an API call, child status not available in parent templates |
| `true` | Read child status — available as `.children.custom["<name>"].status` |
| _(omitted)_ | Auto-detect via REST mapping |

Reference child status in parent templates with `index` (child names are dynamic):

```yaml
status:
  fields:
    - path: childPhase
      value: '{{ (index .children.custom (printf "%s-child" .metadata.name)).status.phase }}'
```

### Owner References (automatic)

Orkestra sets an `ownerReference` on every child CR pointing back to the parent. Deleting the parent cascades to all children automatically — no manual cleanup or `onDelete` hooks needed:

```bash
kubectl delete workspace dev-team
# SecretVault, Deployment, Service — all gone automatically
```

### `when:` conditions

Gate child CR creation on parent spec values. The child is skipped when the condition is false and created when it becomes true on the next reconcile:

```yaml
- apiVersion: platform.example.io/v1alpha1
  kind: CacheCluster
  when:
    - field: spec.cache.enabled
      equals: "true"
  metadata:
    name: "{{ .metadata.name }}-cache"
```

### `reconcile: true` (drift correction)

By default, child CRs are created once. With `reconcile: true`, Orkestra re-applies the child spec on every parent reconcile cycle — any drift is corrected within the resync window:

```yaml
- apiVersion: example.io/v1alpha1
  kind: BackupPolicy
  reconcile: true
  metadata:
    name: "{{ .metadata.name }}-backup"
  spec:
    schedule: "{{ .spec.backup.schedule }}"
```

### `forEach:` fan-out

Create one child CR per element in a parent list field:

```yaml
- apiVersion: storage.example.io/v1alpha1
  kind: Shard
  forEach:
    field: spec.shards
    as: shard
  metadata:
    name: "{{ .metadata.name }}-{{ .shard.name }}"
  spec:
    shardName: "{{ .shard.name }}"
    region: "{{ .shard.region }}"
```

### Motif imports

Package child CR templates into reusable Motifs and import them with `with:` overrides:

```yaml
# In katalog.yaml
crds:
  platform:
    apiTypes: ...
    imports:
      - motif: ./platform-motif.yaml
        with:
          mqReplicas: 3
          searchNodes: 5
```

---

## Missing CRDs

If a target CRD is not yet installed when Orkestra starts, it logs a warning and skips that child gracefully. When the CRD appears (after a `kubectl apply`), Orkestra refreshes its REST mapper automatically — no restart required.

---

## Prerequisites

- Ork CLI: `curl get.orkestra.sh | bash`
- A running Kubernetes cluster (`ork create cluster` works for all examples)
- `kubectl` pointed at the cluster
