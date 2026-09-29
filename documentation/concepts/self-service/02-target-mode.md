# Target Mode

A developer knows the repository and the image tag. They do not know — and should not need to know — `apiVersion`, `kind`, `metadata`, or the difference between `spec` and `labels`. Target mode is the Gateway API request format built for that developer: submit a `target` and flat fields, get a CR back. The gateway builds it.

---

## The problem

Every self-service caller — a browser form, a CI pipeline, a Slack bot — has the same problem: they want to describe what they need, not construct a Kubernetes object.

Full CR mode doesn't solve this — it just moves the Kubernetes object one layer up. The caller still has to know `apiVersion`, still has to know which field is `spec` and which is `metadata.labels`, still has to reconstruct the CR shape by hand on every call. That's fine for `kubectl` and existing integrations. It's the wrong contract for a developer filling out a form.

Target mode hides Kubernetes behind the IDP contract. The platform team defines the fields once. Every caller — form, pipeline, bot — submits the same flat shape, and the gateway does the construction.

---

## How it works

The Katalog declares `serve.target` and `serve.fields`:

```yaml
serve:
  enabled: true
  target: app
  name: '{{ repoSlug .repository }}'
  namespace: '{{ teamName }}-{{ environment }}'
  fields:
    repository:
      label: "Repository"
      type: string
      required: true
    image:
      label: "Container Image"
      type: string
      required: true
```

Callers submit `target` + fields:

```json
POST /api/v1/apply
{
  "target": "app",
  "repository": "myorg/payments-api",
  "image": "ghcr.io/myorg/app:v1.0.0",
  "team": "team-payments",
  "environment": "staging"
}
```

The gateway:

1. Looks up the CRD by `target`
2. Routes fields to `spec`, `metadata.labels`, or `metadata.annotations` based on `serve.fields`, `serve.labels`, and `serve.annotations`
3. Resolves `serve.name` and `serve.namespace`
4. Applies the full CR via SSA

The caller never sees the CR.

---

## Two modes, one API

| Mode | Request format | When to use |
|------|----------------|-------------|
| **Target mode** | `{"target": "...", fields...}` | Self-service callers who don't know Kubernetes |
| **Full CR mode** | `{"apiVersion": "...", "kind": "...", ...}` | Advanced callers, existing clients, `kubectl` compatibility |

```bash
# Target mode — submit fields
curl -X POST /api/v1/apply \
  -d '{"target":"app","repository":"myorg/app","image":"..."}'

# Full CR mode — submit a complete CR
curl -X POST /api/v1/apply \
  -d '{"apiVersion":"platform.myorg.io/v1","kind":"App",...}'
```

Both modes produce the same result. The gateway detects which format you're using based on the presence of `target` or `apiVersion`+`kind`.

---

## The schema contract

Callers discover available targets and fields through the schema API:

```bash
# List all available targets
curl -X GET /api/v1/schema \
  -H "Authorization: Bearer $TOKEN"

# Get fields for a specific target
curl -X GET /api/v1/schema?target=app \
  -H "Authorization: Bearer $TOKEN"
```

The schema API returns a flat list of fields:

```json
{
  "target": "app",
  "title": "Application",
  "fields": {
    "repository": {
      "label": "Repository",
      "type": "string",
      "required": true
    },
    "image": {
      "label": "Container Image",
      "type": "string",
      "required": true
    }
  }
}
```

Callers don't need to know about `spec`, `labels`, or `annotations` — they just see fields.

---

## `serve.target` — the caller-facing identifier

`serve.target` decouples the caller-facing identifier from the Kubernetes `kind`.

```yaml
serve:
  enabled: true
  target: app   # callers use this, not "App" or "apprequests"
```

If omitted, defaults to the lowercased `kind` (e.g., `kind: App` → `target: app`).

`ork validate` ensures targets are unique across the Katalog.

---

## `serve.name` and `serve.namespace`

Target mode resolves `serve.name` and `serve.namespace` server-side, so callers don't need to know them:

```yaml
serve:
  enabled: true
  name: '{{ repoSlug .repository }}'          # → "payments-api"
  namespace: '{{ teamName }}-{{ environment }}' # → "team-payments-staging"
```

Callers never supply `metadata.name` or `metadata.namespace` in target mode.

When `serve.name` is not declared, the caller must supply a name. When `serve.namespace` is not declared on a namespaced CRD, the gateway rejects the request — self-service creation has no way to know where the CR belongs.

---

## Nested fields with `path`

Fields can map to nested locations in the CRD `spec` using `path`:

```yaml
serve:
  fields:
    repository:
      path: app.repository
      label: "Repository"
    cpu:
      path: app.resources.cpu
      label: "CPU Request"
```

Callers submit flat field names:

```json
{
  "target": "app",
  "repository": "myorg/app",
  "cpu": "500m"
}
```

The gateway maps to:

```yaml
spec:
  app:
    repository: myorg/app
    resources:
      cpu: 500m
```

→ [Nested fields with `path` reference](../../reference/schema/02-katalog/20-serve.md#servefieldspath)

---

## Response: `pollUrl` and `payload`

A successful target-mode apply returns:

```json
{
  "accepted": true,
  "name": "payments-api",
  "namespace": "team-payments-staging",
  "kind": "AppRequest",
  "apiVersion": "platform.myorg.io/v1",
  "pollUrl": "/api/v1/resources/AppRequest/team-payments-staging/payments-api?field=status.phase",
  "payload": {
    "phase": "",
    "serviceURL": "https://payments-api.staging.myorg.io",
    "nextSteps": "Waiting for resources to be provisioned..."
  }
}
```

- **`pollUrl`** — where to GET the resource (configurable via `serve.config.response.poll`)
- **`payload`** — the platform team's curated view (`serve.config.response.payload`)

At apply time, `.status` is not yet available. Callers should poll `pollUrl` to see status updates.

→ [`serve.config.response` reference](../../reference/schema/02-katalog/20-serve.md#serveconfigresponse)

---

!!! tip "The one-sentence version"
    The Katalog declares operators; targets declare operational profiles; OperatorBoxes execute those profiles; the Gateway selects the profile; the Runtime provides the shared machinery.

## Target as a unit of runtime execution

A target is not only a routing identifier. Each named target in `serve.target` can carry its own `operatorBox` — a complete set of lifecycle hooks, resource templates, and `preReconcile` gates. When the CR is reconciled, the reconciler selects the `operatorBox` that matches the active target.

This means the same CRD can provision different infrastructure depending on which surface submitted the intent:

```yaml
serve:
  target:
    web:
      primary: true
      operatorBox:
        reconcile:
          onCreate:
            deployments:
              - name: "{{ .metadata.name }}-web"
        preReconcile:
          enqueueGate:
            when:
              - field: "{{ .spec.image }}"
                notEquals: ""
    regional:
      operatorBox:
        reconcile:
          onCreate:
            deployments:
              - name: "{{ .metadata.name }}-{{ .item }}"
                forEach:
                  field: spec.regions
                  as: item
        preReconcile:
          reconcileGate:
            when:
              - field: "{{ len .spec.regions }}"
                notEquals: "0"
```

When a CR switches from one target to another (re-submitted via a different surface), the reconciler detects the change and cleans up resources from the previous target before applying the new box.

→ [Per-target operatorBox schema](../../reference/schema/02-katalog/26-serve-target-operatorbox.md) — gates, surface cleanup, `keepPreviousSurface`

---

## Where to go next

→ [`serve.target` schema reference](../../reference/schema/02-katalog/20-serve.md#servetarget)

→ [`serve.fields` schema reference](../../reference/schema/02-katalog/20-serve.md#servefieldsname)

- [`serve.namespace` reference](../../reference/schema/02-katalog/20-serve.md#servenamespace)

- [Gateway API reference](../../reference/schema/02-katalog/17-gateway-api.md)

- [Additional Fields](01-labels-and-annotations.md)

- [Token Scoping](03-token-scoping.md)