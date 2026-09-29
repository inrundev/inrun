# Reference

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
            X-Request-ID: "{{ .metadata.name }}"
          timeout: 5s
          expectedStatus: 200
          continueOnError: false
          when:
            - field: status.phase
              notEquals: "Ready"
          or: []
          sleep: ""
```

## Field reference

| Field | Required | Default | Description |
|---|---|---|---|
| `name` | yes | — | Identifier for accessing results as `.external.<name>.*`. Must be a valid Go identifier — camelCase, no hyphens. |
| `url` | yes | — | Endpoint URL. Template expressions are resolved against the current CR and any earlier call results in the same reconcile. |
| `method` | no | `GET` | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`. |
| `body` | no | `""` | Request body. Template expressions supported. Sets `Content-Type: application/json` automatically when non-empty. |
| `token` | no | `""` | Bearer token for the `Authorization` header. Use `$ENV_VAR` syntax — expanded via `os.ExpandEnv` at call time. Never put raw secrets here. |
| `headers` | no | `{}` | Additional HTTP headers as a string map. Applied after `Authorization` and `Content-Type`. |
| `timeout` | no | `10s` | Per-call timeout. Go duration format: `"5s"`, `"1m"`, `"500ms"`. Set to a fraction of your `resync:` period. |
| `expectedStatus` | no | `0` | When set, any response with a different status code is treated as a failure. When `0`: 4xx/5xx are errors, 2xx/3xx succeed. |
| `continueOnError` | no | `false` | `false`: failure halts the reconcile and writes `Ready=False` to the CR condition. `true`: failure sets `.error`, the reconcile continues, status fields surface the details. |
| `when` | no | `[]` | AND conditions evaluated before the call runs. If any condition fails, the call is skipped and `.called` is `"false"`. Template expressions in `field:` and in comparison values (`equals:`, `notEquals:`, etc.) are both resolved. |
| `or` | no | `[]` | OR conditions. At least one must pass. Combined with `when:` using AND semantics: `(all when:) AND (any one or:)`. |
| `sleep` | no | `""` | Delay injected before this call runs. Go duration format: `"2s"`. Use to pace sequential calls against rate-limited APIs or to wait for an async side-effect from a prior call. Not a substitute for proper `when:` conditions. |

## Result context

After a call completes, the following fields are available under `.external.<name>`:

| Field | Type | Description |
|---|---|---|
| `.status` | `string` | HTTP status code as a string: `"200"`, `"403"`, `"503"`. Empty when the call failed before receiving a response (network error, timeout). |
| `.body` | `string` | First 4096 bytes of the response body. Truncated silently for larger responses. |
| `.error` | `string` | Error message when the call failed — includes `"expected status 200, got 403"` for status mismatches. Empty on success. |
| `.called` | `string` | `"true"` when the call ran. `"false"` when skipped because `when:`/`or:` conditions failed. |
| `.<key>` | any | When the response body is a valid JSON object, its top-level keys are merged directly into `.external.<name>` and are navigable by dot path. |

If the response is `{ "queue": { "pendingJobs": 8 } }` and the call is named `metrics`, then `.external.metrics.queue.pendingJobs` resolves to `8` — in template expressions and in `field:` conditions.

`.body` is always present alongside the parsed fields. If the body is not valid JSON, only the four fixed fields above are set.

Access in any template expression or condition:

```yaml
# Fixed fields — always available
- field: external.healthCheck.status
  equals: "200"

- field: external.signImage.called
  equals: "true"

value: "{{ .external.appConfig.body }}"

# Parsed JSON fields — available when response is a JSON object
- field: external.metrics.queue.pendingJobs
  greaterThan: "100"

value: "{{ .external.metrics.queue.pendingJobs }}"

# Chaining — later calls can reference earlier results
token: "{{ .external.tokenFetch.body }}"
url: "{{ .external.discovery.body }}/api/v1/{{ .metadata.name }}"
```

### Template expressions in condition comparison values

Both `field:` and the comparison value (`equals:`, `notEquals:`, `prefix:`, `suffix:`, `contains:`) support template expressions. This is how you compare a live CR field against another live field:

```yaml
# Gate the signing call only when spec.image hasn't been signed yet
when:
  - field: status.signedImage
    notEquals: "{{ .spec.image }}"   # resolved against the current CR at eval time

# Gate the Deployment on the image being confirmed safe
when:
  - field: status.signedImage
    equals: "{{ .spec.image }}"      # both sides resolve as strings before comparison
```

Comparison values that contain `{{ }}` are evaluated through the full template engine — note functions, cross-CRD data, and external call results are all available. Non-template values are used as-is.

## Status code prefix matching

The `prefix:` operator on `.external.<name>.status` enables HTTP status class matching without enumerating every code:

```yaml
# Matches any 4xx response — definitive errors, policy decisions
- field: external.signImage.status
  prefix: "4"

# Matches any 5xx response — transient, service unavailable
- field: external.signImage.status
  prefix: "5"
```

This is the standard way to distinguish permanent rejections (4xx — gate future calls with `rejectedImage`) from transient failures (5xx — leave the gate open, retry naturally on the next reconcile).

## Constraints

**Body cap — 4096 bytes.** Large responses are truncated. Filter or paginate on the server side if you need more.

**Sequential execution.** Calls run one at a time in declaration order. A later call can reference an earlier call's result; an earlier call cannot reference a later one.

**camelCase names.** Call names must be valid Go identifiers. Hyphens break template access: `{{ .external.health-check.status }}` does not parse.

**Runs on every reconcile by default.** Unless gated by `when:`, the call fires every time the CR reconciles — including every resync. Design calls to be idempotent, or gate them with `when:` conditions on status fields.

**Token expansion via `os.ExpandEnv`.** The `token:` field expands `$VAR` and `${VAR}` after template resolution. `token: "${{ .spec.tokenEnvVar }}"` does not work — only static `$VAR` references expand. To dynamically select a token, reference an earlier call result directly: `token: "{{ .external.tokenFetch.body }}"`.

**No response streaming.** The operator reads the full response body (up to 4096 bytes) before processing. Long-running or streaming endpoints are not supported.

**`continueOnError: false` + status fields.** When a call fails with `continueOnError: false`, the reconcile returns an error before status fields are evaluated. The call result is available in the resolver — but if you need the failure visible in status (phase, error message), use `continueOnError: true` and enforce the invariant with `when:` conditions on resources instead.
