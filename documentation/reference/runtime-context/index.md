# Runtime Context

Orkestra injects a set of runtime facts into every template expression under the `.ork` namespace. These are constant for the lifetime of the process — set once at startup, available everywhere a `{{ }}` expression is evaluated: status fields, endpoint templates, notes, mutation rules, `when:` conditions.

!!! note "Not notes, not sentinels"
    `.ork.*` facts are not Orkestra notes and not template sentinels. Notes are pure, user-defined functions declared in `notes.functions`. `.ork.*` are process-level constants — some read from the environment at startup (e.g. `inCluster` probes the pod filesystem), but once set they never change for the life of the deployment. Every reconcile, every CR, every template expression sees the same values.

```yaml
notes:
  functions:
    - name: reconcilerEndpoint
      expression: >
        {{ if .ork.inCluster }}
          http://my-reconciler.default.svc.cluster.local:8025/reconcile
        {{ else }}
          http://localhost:8025/reconcile
        {{ end }}
```

```yaml
operatorBox:
  reconcile:
    remote:
      endpoint: "{{ reconcilerEndpoint }}"
```

## Fields

| Key | Type | Description |
|-----|------|-------------|
| `.ork.inCluster` | `bool` | `true` when Orkestra is running inside a Kubernetes pod; `false` during local development. |
| `.ork.namespace` | `string` | Kubernetes namespace Orkestra is deployed in (e.g. `orkestra-system`). |
| `.ork.version` | `string` | Running Orkestra version string (e.g. `v0.7.18`). |

## Discovery

`ork validate --full` prints the runtime context alongside the RBAC and dependency graph:

```text
Runtime context (.ork.*)
  .ork.inCluster    true when Orkestra is running inside a Kubernetes pod
  .ork.namespace    namespace Orkestra is deployed in
  .ork.version      running Orkestra version string
```

## Common patterns

**Endpoint that adapts between local dev and in-cluster:**

```yaml
notes:
  functions:
    - name: myServiceEndpoint
      expression: >
        {{ if .ork.inCluster }}
          http://my-svc.{{ .ork.namespace }}.svc.cluster.local/reconcile
        {{ else }}
          http://localhost:9000/reconcile
        {{ end }}
```

**Status field that exposes the operator namespace:**

```yaml
operatorBox:
  status:
    fields:
      - path: operatorNamespace
        value: "{{ .ork.namespace }}"
```

**When condition that only applies outside the cluster:**

```yaml
when:
  - field: "{{ .ork.inCluster }}"
    equals: false
```

## Where to go next

- [notes](../schema/02-katalog/18-notes.md) — user-defined functions that can wrap `.ork.*` expressions
- [reconcile.remote](../schema/02-katalog/31-reconcile-remote.md) — `endpoint:` template support
