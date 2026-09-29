# status

Orkestra writes status in two layers — one automatic, one declarative.

**Layer 1 — automatic (always on).** After every reconcile Orkestra patches:
- `status.conditions[type=Ready]` — `True` on success, `False` on error
- `status.observedGeneration` — mirrors `metadata.generation`

No declaration required. Every managed CR gets this.

**Layer 2 — declarative fields.** Declared under `operatorBox.status`. Fields with `when:`/`or:` conditions are always evaluated — including when reconcile fails — so status can reflect why. Fields without conditions are only written on successful reconcile.

## Wire format

```yaml
operatorBox:
  emit:
    status:
      conditions: true    # optional — default true

      fields:
        - path: phase
          value: "Running"

        - path: replicas
          type: int
          value: "{{ toInt .spec.replicas }}"

        - path: database.host    # dot-notation → status.database.host
          value: "{{ .spec.host }}"

        - path: endpoint
          value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"

        - path: phase
          value: "Pending"
          when:
            - field: status.phase
              operator: notExists
```

## `conditions`

| Value | Description |
|-------|-------------|
| `true` (default) | Write the standard `Ready` condition and `observedGeneration` after every reconcile. |
| `false` | Opt out. Use only when the CRD's status schema forbids a `conditions` field, or you manage conditions entirely in Go hooks. |

## `fields`

A **list** of field declarations. Resolved in declaration order — later entries win on path conflict.

```yaml
fields:
  - path: <dot.notation.path>     # required
    value: "<string or template>"  # required
    type: string                   # optional
    clearOnFalse: false            # optional
    when:                          # optional — AND conditions
      - ...
    or:                         # optional — OR conditions
      - ...
```

### Field properties

| Field | Required | Description |
|-------|----------|-------------|
| `path` | yes | Dot-notation path relative to `status`. `"phase"` → `status.phase`. `"db.host"` → `status.db.host`. |
| `value` | yes | Value to write. Supports Go template expressions. Static strings skip parsing. |
| `type` | no | Cast the resolved value before writing. Defaults to `string`. |
| `when` | no | List of conditions — **all must pass** (AND). Field is skipped if any fails. |
| `or` | no | List of conditions — **at least one must pass** (OR). |
| `clearOnFalse` | no | When `true` and the `when:`/`or:` condition evaluates to false, write `""` to the field instead of leaving the previous value. Use for transient fields (crash reasons, warning messages) that should disappear when the triggering condition clears. No effect when no conditions are declared. |

When both `when` and `or` are declared, both blocks must pass.

### `type` values

| Value | Written as |
|-------|------------|
| `string` / `str` / `""` (default) | string |
| `int` / `integer` | integer |
| `float` | float64 |
| `bool` / `boolean` | boolean |
| `auto` | inferred from the resolved value |

### Template variables

Values are Go templates evaluated against the full CR object map:

| Expression | Description |
|------------|-------------|
| `{{ .metadata.name }}` | CR name |
| `{{ .metadata.namespace }}` | CR namespace |
| `{{ .spec.* }}` | Any spec field |
| `{{ .status.* }}` | Existing status fields |
| `{{ .children.deployment }}` | Child Deployment status map |
| `{{ .children.statefulset }}` | Child StatefulSet status map |
| `{{ .children.service }}` | Child Service status map |
| `{{ .children.job }}` | Child Job status map |
| `{{ .children.custom }}` | Child Custom Resource status map |

### Kubernetes helper functions

Functions from the Orkestra note library useful in status values:

**Replicas**

| Function | Returns | Description |
|----------|---------|-------------|
| `allReplicasReady .children.deployment` | bool | `true` when desired == ready |
| `readyReplicas .children.deployment` | int | Number of ready replicas |
| `desiredReplicas .children.deployment` | int | Desired replica count |
| `availableReplicas .children.deployment` | int | Available (not just ready) replicas |
| `updatedReplicas .children.deployment` | int | Replicas on current pod template |
| `rolloutComplete .children.deployment` | bool | `true` when rollout is fully done |

**Services**

| Function | Returns | Description |
|----------|---------|-------------|
| `serviceClusterIP .children.service` | string | Cluster IP of the service |
| `serviceNodePort .children.service "portName"` | int | NodePort for the named port |
| `serviceLoadBalancerIP .children.service` | string | Load balancer IP |
| `serviceLoadBalancerHost .children.service` | string | Load balancer hostname |
| `endpointsReady .children.endpointslice` | bool | `true` when endpoints are ready |

**Jobs**

| Function | Returns | Description |
|----------|---------|-------------|
| `jobSucceeded .children.job` | bool | `true` when job completed successfully |
| `jobFailed .children.job` | bool | `true` when job failed |

**Type casting**

| Function | Description |
|----------|-------------|
| `toInt .spec.replicas` | Cast to integer |
| `toFloat .spec.ratio` | Cast to float |
| `toBool .spec.enabled` | Cast to bool |
| `toString .spec.count` | Cast to string |

### `when` and `or` conditions

Each condition targets a dot-notation field path and applies an operator.

```yaml
when:
  - field: status.phase      # required — dot-path into the CR object
    equals: "Running"        # shorthand operators (see below)

  - field: spec.replicas
    operator: gt             # explicit operator form
    value: "0"
```

`status.fields[].when`/`or` use the exact same `Condition` type, operators, and shorthand fields as resource-template `when:`/`or:` — see [when/or conditions § Operators](06-when-conditions.md#operators) for the full list (`equals`, `contains`, `prefix`/`suffix`, `regex`, `exists`/`notExists`, `gt`/`lt`/`gte`/`lte`/`between`, `in`/`notIn`, the `typeOf` family, and their shorthand names). Absent field is treated as `0` for numeric comparisons.

## Example: declarative state machine

```yaml
status:
  fields:
    - path: phase
      value: "Pending"
      when:
        - field: status.phase
          operator: notExists

    - path: phase
      value: "Running"
      when:
        - field: status.phase
          equals: "Pending"
        - field: children.deployment.status.readyReplicas
          operator: gt
          value: "0"

    - path: phase
      value: "Succeeded"
      when:
        - field: status.phase
          equals: "Running"
        - field: children.job.status.succeeded
          operator: gt
          value: "0"

    - path: ready
      type: bool
      value: "{{ allReplicasReady .children.deployment }}"

    - path: endpoint
      value: "{{ serviceLoadBalancerHost .children.service }}"
```

---
