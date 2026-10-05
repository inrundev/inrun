# 04-configmap-copy

A cluster-scoped `ConfigMapDistribution` copies the `app-config` ConfigMap
from `platform` into each target namespace and keeps the copies in sync.

### 1. Simulate

No cluster needed.

```bash
inrun simulate
```

### 2. Run

Applies `manifests/setup.yaml` first: the `platform` namespace and the source ConfigMap.

```bash
inrun
```

### 3. Check a copy

In a second terminal.

```bash
kubectl get configmap app-config -n team-alpha -o yaml
```

### 4. Clean up

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

### 5. E2E

Runs the same checks in a kind cluster.

```bash
inrun e2e
```
