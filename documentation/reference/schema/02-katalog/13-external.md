# external

The `external:` list under `onReconcile:` declares HTTP calls to make before any resource group is processed. Results are injected into the template resolver under `.external.<name>.*` and are available in every subsequent template expression, `when:` condition, and status field.

Calls run sequentially in declaration order. The resolver is updated after each call — later calls can reference earlier results via template expressions in their `url:`, `token:`, or `body:` fields.

## Wire format

```yaml
operatorBox:
  reconcile:
    onReconcile:
      external:
        - name: healthCheck
          url: "{{ .spec.serviceUrl }}/health"
          method: GET
          body: ""
          token: "$API_TOKEN"
          headers:
            X-Request-Source: orkestra
          timeout: 5s
          expectedStatus: 200
          continueOnError: false
          when:
            - field: status.phase
              notEquals: "Ready"
          sleep: ""
```

## Fields

| Field | Required | Default | Description |
|---|---|---|---|
| `name` | yes | — | Result identifier. Must be a valid Go identifier (camelCase). Used as `{{ .external.<name>.status }}`. |
| `url` | yes | — | Endpoint URL. Template expressions are resolved against the current CR. |
| `method` | no | `GET` | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`. |
| `body` | no | `""` | Request body. Template expressions supported. Sets `Content-Type: application/json` when non-empty. |
| `token` | no | `""` | Bearer token. Use `$ENV_VAR` syntax — expanded via `os.ExpandEnv`. Never put raw secrets here. |
| `headers` | no | `{}` | Additional HTTP headers as `map[string]string`. |
| `timeout` | no | `10s` | Per-call deadline. Go duration: `"5s"`, `"1m"`, `"500ms"`. |
| `expectedStatus` | no | `0` | When set: any other status code is a failure. When `0`: `4xx`/`5xx` is a failure, `2xx` succeeds. |
| `continueOnError` | no | `false` | `false`: failure halts reconcile, writes `Ready=False`. `true`: failure logged, reconcile continues. |
| `when` | no | `[]` | AND gate conditions. If any fail, the call is skipped and `.called = "false"`. |
| `or` | no | `[]` | OR gate conditions. At least one must pass. Combined with `when:` using AND semantics. |
| `sleep` | no | `""` | Delay before this call. Go duration. For development and sequencing async side-effects — not for production rate limiting. |
| `fires.reconcile` | no | `true` | When `false`, the call is skipped during reconcile — it only runs at admission time. Applies when the call is declared under `validation.external` or `mutation.external`. No effect on `onReconcile.external` calls. |
| `include` | no | — | Path to a YAML file with a top-level `calls:` list. When set, this entry is replaced in-place by the listed calls. Resolved relative to the katalog file. Cleared after expansion. |
| `retryBackoff` | no | — | Retry this specific call with exponential backoff before returning an error. Shorthand (`"2s"`) sets `initial` only; full form: `initial`, `max`, `multiplier`, `maxAttempts`. See [retry backoff](../../concepts/operatorbox/09-retry-backoff.md). |

## Result context

| Field | Description |
|---|---|
| `.external.<name>.status` | HTTP status code string (`"200"`, `"503"`). Empty on pre-response failure. |
| `.external.<name>.body` | First 4096 bytes of response body. |
| `.external.<name>.error` | Error message on failure; `""` on success. |
| `.external.<name>.called` | `"true"` when the call ran; `"false"` when skipped by `when:`/`or:`. |
| `.external.<name>.<key>` | When the response body is a valid JSON object, its top-level keys are merged in and navigable by dot path. |

If the response is `{ "queue": { "pendingJobs": 8 } }` and the call is named `metrics`, then `external.metrics.queue.pendingJobs` is directly usable in `field:` conditions and `{{ .external.metrics.queue.pendingJobs }}` in template expressions. `.body` is always present alongside parsed fields.

## Access pattern

```yaml
# Gate a resource on a successful call
deployments:
  - name: "{{ .metadata.name }}"
    when:
      - field: external.healthCheck.status
        equals: "200"

# Embed the body in a ConfigMap
configMaps:
  - name: "{{ .metadata.name }}-config"
    data:
      config.json: "{{ .external.appConfig.body }}"

# Use a previous call's result in the next call's token
external:
  - name: tokenFetch
    url: "{{ .spec.authUrl }}/token"
    method: POST

  - name: protectedCall
    url: "{{ .spec.serviceUrl }}/resource"
    token: "{{ .external.tokenFetch.body }}"
```

## `include:` in external lists

When the `external:` list grows long, individual entries can delegate to a shared file:

```yaml
onReconcile:
  external:
    - include: ./shared/auth-calls.yaml
    - name: healthCheck
      url: "{{ .spec.serviceUrl }}/health"
```

`./shared/auth-calls.yaml`:

```yaml
calls:
  - name: tokenFetch
    url: "{{ .spec.authUrl }}/token"
    method: POST
  - name: featureFlags
    url: "{{ .spec.flagsUrl }}/flags"
    continueOnError: true
```

The include entry is replaced in-place by the `calls:` list. The path is resolved relative to the katalog file's directory. Works in `onReconcile.external`, `onCreate.external`, `validation.external`, and `mutation.external`.

## Placement

External calls can be declared in three locations:

| Location | When it fires |
|---|---|
| `onReconcile.external` | Every reconcile cycle, before resource groups are processed |
| `validation.external` | Before validation rules — at admission (always) and at reconcile (unless `fires.reconcile: false`) |
| `mutation.external` | Before mutation rules — at admission (always) and at reconcile (unless `fires.reconcile: false`) |

`fires.reconcile: false` is useful when an external call is expensive or irrelevant after the CR is persisted — for example, a pre-admission health check that only makes sense at `kubectl apply` time.

```yaml
admission:
  validation:
    external:
      - name: healthCheck
        url: "{{ .spec.serviceUrl }}/health"
        expectedStatus: 200
        continueOnError: true
        fires:
          reconcile: false   # checked once at apply — not repeated every resync
    rules:
      - field: "{{ .external.healthCheck.status }}"
        equals: "200"
        action: deny
        message: "health check failed — deployment blocked"
```

## Constraints

- **Body cap**: 4096 bytes — larger responses are truncated silently.
- **Sequential**: no parallel execution; declaration order is execution order.
- **camelCase names**: hyphens in `name` break template access.
- **Every reconcile**: calls run on every reconcile unless gated by `when:`.
- **Token expansion**: `$VAR` and `${VAR}` only — template expressions in `token:` resolve first, then env expansion applies.

## Example

```yaml
operatorBox:
  reconcile:
    onReconcile:
      external:
        - name: healthCheck
          url: "{{ .spec.serviceUrl }}/health"
          expectedStatus: 200
          continueOnError: true
          timeout: 5s

      deployments:
        - name: "{{ .metadata.name }}"
          image: "{{ .spec.image }}"
          when:
            - field: external.healthCheck.status
              equals: "200"
  emit:
    status:
      fields:
        - path: phase
          value: "Degraded"
          when:
            - field: external.healthCheck.status
              notEquals: "200"
        - path: phase
          value: "Ready"
          when:
            - field: external.healthCheck.status
              equals: "200"
            - field: "{{ allReplicasReady .children.deployment }}"
              equals: "true"
        - path: lastHealthCheck
          value: "{{ .external.healthCheck.status }}"
```

See the [External concept doc](../../../concepts/operatorbox/07-external/index.md) for patterns and best practices.
