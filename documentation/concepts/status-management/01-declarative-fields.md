# Declarative Status Fields

Declare status fields in the Katalog. Values support the same Go template expressions as `onCreate` templates — resolved against the live CR at reconcile time.

```yaml
operatorBox:
  emit:
    status:
      fields:
        - path: phase
          value: "Running"

        - path: observedReplicas
          value: "{{ .spec.replicas }}"

        - path: endpoint
          value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"

        - path: version
          value: "{{ .spec.version }}"

        - path: database.host        # nested — becomes status.database.host
          value: "{{ .spec.host }}"

        - path: database.port
          value: "{{ .spec.port }}"
```

After a successful reconcile:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
  observedGeneration: 1
  phase: Running
  observedReplicas: "2"
  endpoint: my-site.orkestra.svc.cluster.local
  version: "1.25"
  database:
    host: db.platform.svc
    port: "5432"
```

---

## `when:` and `or:`

Gate a field on conditions — the same condition engine used in resource templates and validation rules.

```yaml
status:
  fields:
    - path: phase
      value: "Active"
      when:
        - field: external.healthCheck.status
          equals: "200"
        - field: "{{ allReplicasReady .children.deployment }}"
          equals: "true"

    - path: phase
      value: "Degraded"
      when:
        - field: external.healthCheck.status
          notEquals: "200"

    - path: phase
      value: "Pending"
      or:
        - field: external.healthCheck.called
          equals: "false"
        - field: "{{ allReplicasReady .children.deployment }}"
          equals: "false"
```

`when:` requires all conditions to pass (AND). `or:` requires at least one to pass (OR). When both are declared, both blocks must pass.

A path can appear multiple times with different conditions — the first matching entry wins. Use this to build declarative state machines.

---

## `include:`

When the field list grows long it can be split into a separate file:

```yaml
operatorBox:
  emit:
    status:
      include: ./status/fields.yaml
      fields:
        - path: version           # appended after included fields
          value: "{{ .spec.version }}"
```

`./status/fields.yaml`:

```yaml
fields:
  - path: phase
    value: "Active"
    when:
      - field: "{{ allReplicasReady .children.deployment }}"
        equals: "true"
  - path: phase
    value: "Degraded"
    when:
      - field: "{{ allReplicasReady .children.deployment }}"
        equals: "false"
```

Included fields come first. Inline `fields:` append after. The path is resolved relative to the katalog file's directory.

---

## Rules

**Paths are relative to `status`.** `phase` writes to `status.phase`. `database.host` writes to `status.database.host`. Dot-notation works at any depth.

**Unconditional fields are only written on successful reconcile.** A field with no `when:` or `or:` is skipped when reconcile fails — writing it on error would produce misleading status (e.g. `phase: Active` while the CR is denied).

**Conditional fields always evaluate.** A field with `when:` or `or:` is evaluated on both success and failure. This is what allows status to reflect *why* reconcile failed — for example, surfacing an external health check result or the denial reason as `phase: Degraded`.

---

## CRD schema tip

!!! tip "Document your status fields in the CRD schema"
    Declare the status fields in the CRD's OpenAPIV3Schema to enable `kubectl` validation. Use `x-kubernetes-preserve-unknown-fields: true` to accept any field without enumerating every one:
    ```yaml
    status:
      type: object
      x-kubernetes-preserve-unknown-fields: true
    ```
