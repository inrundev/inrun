# Resource Conditions

Attach a `when:` or `or:` block to any resource declaration under `onCreate`, `onReconcile`, or `onDelete`. Orkestra evaluates it before creating, updating, or skipping that resource.

---

## Basic usage

```yaml
operatorBox:
  reconcile:
    onCreate:
      services:
        - name: "{{ .metadata.name }}-public"
          type: LoadBalancer
          port: "80"
          targetPort: "{{ .spec.port }}"
          reconcile: true
          when:
            - field: spec.exposePublicly
              equals: "true"
  crdFile: my-operator-crd.yaml
```

If the CR has `spec.exposePublicly: "false"`, the service is skipped. No error. No partial state.

---

## Multiple conditions (AND)

All conditions in `when:` must pass:

```yaml
when:
  - field: spec.environment
    equals: "production"
  - field: spec.replicas
    gt: "2"
```

Both must be true for the resource to be created.

---

## OR conditions with `or:`

```yaml
or:
  - field: spec.environment
    equals: "production"
  - field: spec.forceExpose
    equals: "true"
```

At least one must be true.

---

## Optional fields

Use `exists` to gate on fields that may not be present:

```yaml
configMaps:
  - name: "{{ .metadata.name }}-config"
    data:
      config.yaml: "{{ .spec.config }}"
    reconcile: true
    when:
      - field: spec.config
        exists: true
```

If `spec.config` is missing, the ConfigMap is skipped.

---

## Feature flags

```yaml
deployments:
  - name: "{{ .metadata.name }}-worker"
    image: "{{ .spec.workerImage }}"
    replicas: "{{ .spec.workerReplicas }}"
    reconcile: true
    when:
      - field: spec.features.workers
        equals: "enabled"
```

---

## What skipping means

A skipped resource is a clean no-op. It does not count as an error, does not requeue the CR, and does not delete a previously-created resource. If a resource was created on a prior reconcile and the condition later fails, Orkestra stops reconciling it — but does not remove it. Use `onDelete` for explicit teardown.
