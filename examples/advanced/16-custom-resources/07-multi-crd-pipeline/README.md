# 07 — Multi-CRD Pipeline (Motif + Child Fan-Out)

A `Pipeline` CRD orchestrates a full CI/CD workflow by creating three specialist child CRDs — `Loader`, `Processor`, and `Auditor` — each managed by its own dedicated controller. A **Motif** captures the parameterised template so the same pattern can be re-used for different pipelines.

This example brings together nearly every concept from the earlier examples:
- Motif-driven templating (example 06)
- `hasStatus: true` on every child so the parent can track their phase (example 02)
- Multiple child CRs per parent (new here)
- Child CRD controllers defined in the same Katalog as the parent

---

## What This Shows

1. A `Pipeline` CR is applied.
2. Orkestra expands `pipeline-motif.yaml` with the supplied `with:` overrides.
3. Three child CRs are created atomically: `{name}-loader`, `{name}-processor`, `{name}-auditor`.
4. The `loader`, `processor`, and `auditor` controllers in the same Katalog pick up their respective CRs and create Deployments + Services.
5. Each child reports its `phase` back to the parent's resolver context (because `hasStatus: true`).
6. Deleting the Pipeline cascades to all three children via owner references.

---

## New Concepts Introduced

### Multiple child CRs per parent

A single `onCreate.custom` block can declare as many child CRs as needed. They are created in declaration order, in the same reconcile pass:

```yaml
operatorBox:
  reconcile:
    onCreate:
      custom:
        - apiVersion: autoscale.orkestra.io/v1alpha1
          kind: Loader
          metadata:
            name: "{{ .metadata.name }}-loader"
            ...
        - apiVersion: autoscale.orkestra.io/v1alpha1
          kind: Processor
          metadata:
            name: "{{ .metadata.name }}-processor"
            ...
        - apiVersion: autoscale.orkestra.io/v1alpha1
          kind: Auditor
          metadata:
            name: "{{ .metadata.name }}-auditor"
            ...
```

### Motif import with `with:` overrides

The `imports:` block in the Katalog binds the Motif to a specific CRD and injects default input values:

```yaml
crds:
  pipeline:
    ...
    imports:
      - motif: ./pipeline-motif.yaml
        with:
          processorImage: python:3.12
          auditMode: Strict
          replicas: 3
```

Each individual Pipeline CR can override inputs via its own `spec` fields after template resolution. The Motif defines the schema (`inputs:`) and the Katalog sets cluster-wide defaults.

### Child controllers in the same Katalog

The `Loader`, `Processor`, and `Auditor` controllers are all declared in `katalog.yaml` alongside `Pipeline`. When Orkestra starts, it spins up a goroutine per CRD. The Pipeline controller creates the child CRs; the child controllers react to them independently — they do not know they were created by Pipeline.

### `hasStatus: true` for status propagation

Each child CR has `hasStatus: true`, so Orkestra reads the child's `status` into the parent's template resolver context:

```yaml
# In pipeline-motif.yaml
- apiVersion: autoscale.orkestra.io/v1alpha1
  kind: Loader
  ...
  hasStatus: true
```

In parent templates you can then reference:

```yaml
status:
  fields:
    - path: loaderPhase
      value: "{{ index .children.custom 0 \"status\" \"phase\" }}"
```

---

## File Layout

```
07-multi-crd-pipeline/
├── katalog.yaml          # Defines Pipeline + Loader + Processor + Auditor operators
├── pipeline-motif.yaml   # Parameterised child CR template (imported by Pipeline)
├── crd-pipeline.yaml     # Pipeline CRD schema
├── crd-loader.yaml       # Loader CRD schema
├── crd-processor.yaml    # Processor CRD schema
├── crd-auditor.yaml      # Auditor CRD schema
├── cr-pipeline.yaml      # Sample Pipeline CR instance
└── cleanup.sh
```

---

## Prerequisites

- `kubectl` configured to a running cluster (Kind works)
- Ork CLI:
  ```bash
  curl get.orkestra.sh | bash
  ```

---

## Run the Example

### 1. Apply all CRDs

```bash
kubectl apply -f crd-pipeline.yaml
kubectl apply -f crd-loader.yaml
kubectl apply -f crd-processor.yaml
kubectl apply -f crd-auditor.yaml
```

### 2. Start the runtime

```bash
ork run --dev
```

You should see four controllers start — `Pipeline`, `Loader`, `Processor`, `Auditor` — all in the same process.

### 3. Apply the Pipeline CR

```bash
kubectl apply -f cr-pipeline.yaml
```

### 4. Observe the cascade

```bash
# Parent is up
kubectl get pipelines -n default

# Three children created automatically
kubectl get loaders,processors,auditors -n default

# Loader/Processor also created Deployments + Services
kubectl get deployments,services -n default
```

Expected:

```
NAME         LANGUAGE   AGE
webapp-go    go         12s

NAME               IMAGE               PHASE     AGE
webapp-go-loader   nginx:stable-alpine Running   10s

NAME                    IMAGE         WORKERS   AUTOSCALE   PHASE     AGE
webapp-go-processor     python:3.12   0                     Running   10s

NAME                  AUDITMODE   PHASE     AGE
webapp-go-auditor     Strict                Running   10s
```

### 5. Verify owner references

```bash
kubectl get loader webapp-go-loader -n default \
  -o jsonpath='{.metadata.ownerReferences[0].name}'
# webapp-go
```

### 6. Delete and watch cascade

```bash
kubectl delete pipeline webapp-go -n default
kubectl get loaders,processors,auditors -n default
# Expected: No resources found
```

---

## What to Observe

- All three child CRs are created in a single reconcile cycle.
- Each child's `phase` is available in the Pipeline's template context when `hasStatus: true`.
- Deleting the Pipeline removes all three children in one shot via Kubernetes owner reference garbage collection.
- The Motif `with:` overrides flow through to child CR specs (`processorImage: python:3.12` → `Processor.spec.image`).
- The `Loader` defaults to the global `image` input because `loaderImage` was not overridden.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
