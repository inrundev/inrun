# reconcile.remote

`reconcile.remote:` wires a remote HTTP endpoint as the reconciler for a CRD. Instead of running logic inside the runtime binary, Orkestra POSTs a `PreparedRequest` to the declared endpoint after every watch event. The external service returns a `RemoteReconcileResult`; Orkestra applies the result and manages the full operator lifecycle on its behalf — informer, queue, workers, backoff, health, RBAC.

Use this when reconcile logic lives in a separate service (e.g. a sidecar, a mesh-external API, or a language runtime that Orkestra does not host).

## Wire format

```yaml
operatorBox:
  reconcile:
    default: false
    remote:
      endpoint: "https://my-reconciler.internal/reconcile"
      type: http
      timeout: 30s
      args:
        logLevel: "{{ .spec.logLevel }}"
        environment: "{{ .spec.environment }}"
      payload:
        object:
          exclude:
            - metadata.managedFields
            - metadata.annotations
        children:
          enabled: true
          exclude:
            - metadata.managedFields
      managedResources:
        - group: apps
          plural: deployments
        - group: ""
          plural: services
      auth:
        secretRef:
          name: my-reconciler-token
          key: token
        header: "Authorization"
```

## Fields

### `reconcile.remote:`

| Field | Required | Default | Description |
|---|---|---|---|
| `endpoint` | yes | — | URL that Orkestra POSTs `PreparedRequest` to. Template expressions are supported and evaluated per-reconcile against the CR's resolver context. Use `.ork.inCluster` to switch between local dev and in-cluster: `'{{ if .ork.inCluster }}http://svc.ns.svc.cluster.local/reconcile{{ else }}http://localhost:8025/reconcile{{ end }}'`. See [runtime context](../../runtime-context/index.md). |
| `type` | no | `http` | Wire protocol. `http` is the only supported value; `grpc` is reserved for a future release. |
| `timeout` | no | `30s` | Per-call deadline. Go duration string: `"5s"`, `"1m"`. |
| `args` | no | — | Template expressions evaluated against the CR at reconcile time, injected as a flat `args` map in `PreparedRequest`. Useful for derived labels, computed flags, or any value the reconciler needs without parsing the full object. |
| `payload` | no | — | Controls what is included in the outbound payload. See [`payload:` fields](#payload-fields) below. |
| `managedResources` | no | `[]` | Child resource types this reconciler creates. Used for RBAC generation — same semantics as `constructor.managedResources`. |
| `auth` | no | — | Credential for the outbound request. See [`auth:` fields](#auth-fields) below. |

### `payload:` fields

| Field | Required | Default | Description |
|---|---|---|---|
| `payload.object.exclude` | no | `[]` | Dot-notation paths stripped from the CR object before dispatch (e.g. `metadata.managedFields`). Applied to every request. |
| `payload.children.enabled` | no | `false` | When `true`, child resources from the previous reconcile cycle are injected into `prepared.children`. Keyed by lowercase kind then name. |
| `payload.children.exclude` | no | `[]` | Dot-notation paths stripped from every child resource. Acts as the root exclude list when no per-resource override is set. |
| `payload.children.resources` | no | — | Per-resource child config. Keys are resource identifiers (same as `managedResources`). Each entry may set its own `exclude` list. A `null` value inherits the root `exclude`. |

### `auth:` fields

| Field | Required | Default | Description |
|---|---|---|---|
| `auth.secretRef.name` | yes (if secretRef) | — | Kubernetes Secret name. Template expressions are supported (`{{ .metadata.name }}-token`). Static names are scoped in generated RBAC (`resourceNames:`); template names produce an unscoped `get` rule. |
| `auth.secretRef.key` | yes (if secretRef) | — | Data key within the Secret whose value is injected as the credential. |
| `auth.secretRef.namespace` | no | operator namespace | Secret namespace. Defaults to Orkestra's own namespace. |
| `auth.env` | no | — | Environment variable name (without `$`). Read from the operator pod's environment. |
| `auth.header` | no | `Authorization` | HTTP header the credential is injected into as `Bearer <value>`. Set to `X-Api-Key` or similar for non-Bearer schemes. |

## Prerequisites

`reconcile.remote:` requires `default: false`. Without it the validator rejects the config.

```yaml
operatorBox:
  reconcile:
    default: false   # required
    remote:
      endpoint: "..."
```

## RBAC

Orkestra generates RBAC rules for `managedResources` and `auth.secretRef` automatically:

- Each entry in `managedResources` produces a `get/list/watch/create/update/patch/delete` rule for that resource group.
- Static `auth.secretRef.name` values are scoped to `resourceNames:` in the generated `ClusterRole`.
- Template `auth.secretRef.name` values (those containing `{{`) produce an unscoped `secrets: get` rule since the name is not known at build time.

## simulate

Remote reconcilers cannot be simulated. `ork simulate` prints a note and skips any CRD that declares `reconcile.remote:`.

```text
  note: remote reconciler — skipping (dispatches to an external endpoint at runtime)
```

## PreparedRequest shape

The payload Orkestra POSTs to `endpoint` is a JSON-encoded request body:

```json
{
  "key":      "default/my-webapp",
  "gvk":      { "Group": "myorg.io", "Version": "v1", "Kind": "MyKind" },
  "object":   { ... },
  "args":     { "logLevel": "info", "environment": "production" },
  "prepared": { ... }
}
```

| Field | Description |
|---|---|
| `key` | `namespace/name` of the reconciled CR. |
| `gvk` | GroupVersionKind of the reconciled resource. |
| `object` | The current CR as a plain JSON object. Paths listed in `payload.object.exclude` are stripped before dispatch. |
| `args` | Flat key/value map of template expressions from `reconcile.remote.args`, evaluated against the CR. Present only when `args` is declared. |
| `prepared` | The full PreparedRequest as enriched by Orkestra — resolver context, cross-CRD state, and validation result. When `payload.children.enabled` is true, `prepared.children` contains child resources from the previous cycle, keyed by lowercase kind then name. |

## RemoteReconcileResult shape

The endpoint must respond with `200 OK` and a JSON body:

```json
{
  "result":       "ok",
  "requeueAfter": "",
  "error":        "",
  "status":       { "phase": "Ready" },
  "resources": [
    {
      "type":   "deployment",
      "fields": { "name": "my-webapp", "image": "nginx:latest", "replicas": 2, "port": 8080 }
    },
    {
      "type":   "service",
      "fields": { "name": "my-webapp-svc", "port": 80, "targetPort": 8080 }
    }
  ]
}
```

| Field | Description |
|---|---|
| `result` | Required. One of `"ok"` (done), `"requeue"` (re-enqueue after `requeueAfter`), or `"error"` (failure). |
| `requeueAfter` | Go duration string. Used when `result` is `"requeue"`: re-enqueue after this delay (e.g. `"60s"`). |
| `error` | Human-readable error message. Non-empty triggers backoff retry and sets `Ready=False`. |
| `status` | Optional map of fields to patch onto the CR's status. Remote fields win on conflict with any katalog `emit.status` patch. Omit or set `null` to leave status unchanged. |
| `resources` | Optional list of resources to apply via SSA. Each entry is either an **intent form** (`type` + `fields`) or a **full object** (`apiVersion`, `kind`, and `metadata.name`). Only types declared in `managedResources` are accepted — any undeclared type causes an immediate error before any resource is applied. Orkestra sets owner references for same-namespace resources. |
| `forceConflict` | Per-resource `bool`. When `true`, SSA is applied with `force: true` — Orkestra takes ownership of any conflicting fields. Per-resource value overrides the CRD-level setting; when omitted, falls back to the CRD-level setting, then defaults to `true`. |

### Intent form

The intent form lets the reconciler return a named type and a flat field map. Orkestra constructs the full Kubernetes object. Optionally set `forceConflict` to control SSA field ownership for that resource.

```json
{ "type": "deployment", "fields": { "name": "my-app", "image": "nginx:latest", "replicas": 2 }, "forceConflict": true }
```

Supported types:

| Type | Kubernetes kind |
|---|---|
| `configmap` | ConfigMap |
| `cronjob` | CronJob |
| `custom` | custom resource (requires `kind` in `fields`) |
| `deployment` | Deployment |
| `ingress` | Ingress |
| `job` | Job |
| `secret` | Secret |
| `serviceaccount` | ServiceAccount |
| `service` | Service |
| `statefulset` | StatefulSet |

For built-in types (`deployment`, `service`, etc.) the `kind` field is inferred from `type` — omit it. For `custom`, `kind` must be set in `fields` since there is no single Kubernetes kind to infer.

### Full object form

When more control is needed, return a complete Kubernetes object:

```json
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": { "name": "my-app" },
  "spec": { ... }
}
```

The type must still be declared in `managedResources`.
