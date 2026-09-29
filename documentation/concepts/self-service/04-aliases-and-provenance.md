# Aliases and Intent Provenance

A serve target is a stable caller-facing name for a CRD. One target, one surface. That works for most cases. But some CRDs serve genuinely different audiences — a CI pipeline that needs a fast preview, a platform team that needs full create/delete access, a developer who only needs read access to check status. Each audience wants a different contract: different permissions, different response shape, different behavior in the reconciler.

You could solve this with multiple CRDs — one per audience. But the actual Kubernetes resource is the same. The CR is the same. The operator is the same. What's different is the *surface* through which intent arrives and the *context* that surface brings.

Aliases name those surfaces.

---

## What an alias is

An alias is a named entry point alongside the primary serve target. It shares the same CRD, the same operator, and the same Kubernetes resource underneath — but it can restrict which tokens are valid for it, and shape the response differently.

All targets — primary and aliases — live in one `serve.target` map. The entry with `primary: true` is the primary surface; all other entries are aliases:

```yaml
serve:
  target:
    apifixture:
      primary: true
    preview:
      tokens:
        control-center:
          permissions:
            global: [get, list]
    internal:
      tokens:
        platform-team:
          permissions:
            global: ["*"]
```

Callers use aliases the same way they use the primary target — by name in `POST /api/v1/apply`:

```bash
curl -X POST /api/v1/apply \
  -H "Authorization: Bearer $CONTROL_CENTER_TOKEN" \
  -d '{"target": "preview", "name": "my-service", ...}'
```

The `preview` alias is now a stable delivery surface. Callers don't need to know the underlying CRD kind, the namespace, or the token permissions — the alias encapsulates all of that.

---

## Token resolution

Token access resolves through a chain, most specific first:

1. **Entry tokens** — when the target entry declares `tokens:`, only those entries are checked. A token absent from the entry token map is denied, even if it is valid at the gateway level or the CRD level.
2. **CRD tokens** — when the entry declares no `tokens:`, the CRD-level `serve.tokens` restrictions apply.
3. **Allow all** — when neither level declares restrictions, any valid gateway token is allowed.

This means aliases can *narrow* access below the CRD level, but not *widen* it. A CI token with read-only CRD permissions cannot get write access through an alias.

```yaml
serve:
  tokens:
    control-center:
      permissions:
        global: ["*"]
    ci-pipeline:
      permissions:
        resources: [get, list]

  target:
    apifixture:
      primary: true
    preview:
      # only control-center — ci-pipeline denied here even though it has CRD access
      tokens:
        control-center:
          permissions:
            global: [get, list]
```

---

## Intent provenance

When a CR is applied through the gateway, three annotations are stamped on the CR by the apply handler:

| Annotation | Value |
|---|---|
| `orkestra.orkspace.io/serve-target` | The primary target name — always set |
| `orkestra.orkspace.io/serve-alias` | The alias name, or `""` for the primary target |
| `orkestra.orkspace.io/serve-source` | The verified OIDC `sub` claim of the caller, or `""` for static token auth |

These are permanent. They survive updates and resyncs. They say: *this CR arrived through this surface, from this caller identity, and carried that context forward.*

Built-in notes expose them in templates:

| Note | Returns |
|------|---------|
| `getServeTarget .` | Primary target name |
| `getServeAlias .` | Alias name — `""` for primary target |
| `getServeSource .` | Delivery source — `""` for direct Gateway API calls |
| `hasServeTarget .` | `true` when submitted via the Gateway API |
| `hasServeAlias .` | `true` when a named alias was used |
| `hasServeSource .` | `true` when a webhook source integration was used |
| `isDirectApply .` | `true` when none of the three annotations are present — raw `kubectl` or CI direct apply |

---

## Routing at admission and reconcile time

`getServeAlias` and `getServeTarget` are available in `when:` conditions at both admission time and reconcile time. The same note functions, the same syntax — the only difference is when they fire.

When the same alias check appears across validation, mutation, and reconcile rules, declare it once as a user-defined note:

```yaml
notes:
  functions:
    - name: isPreview
      expression: '{{ eq (getServeAlias .) "preview" }}'
    - name: isPrimary
      expression: '{{ eq (getServeAlias .) "" }}'
```

`isDirectApply` is a built-in note — no declaration needed. It returns `true` when none of the three provenance annotations are present, meaning the CR arrived via `kubectl` or CI direct apply rather than through any gateway surface.

Then reference the notes by name instead of repeating the expressions.

**At admission time** (validation and mutation rules) — the gateway stamps the annotations on the CR before the SSA patch reaches the API server. The webhook sees them on every create and update. Use this to gate defaults and enforce policies per surface:

```yaml
admission:
  validation:
    rules:
      - field: spec.replicas
        lessThanOrEqualTo: 10
        message: "preview environments are capped at 10 replicas"
        action: deny
        when:
          - field: '{{ isPreview }}'
            equals: "true"
      # direct kubectl applies must use the internal registry — no gateway classification to rely on
      - field: spec.image
        prefix: "myorg/"
        message: "direct applies must use the internal registry"
        action: deny
        when:
          - field: '{{ isDirectApply . }}'
            equals: "true"
  mutation:
    rules:
      - field: metadata.labels.ttl
        default: "24h"
        when:
          - field: '{{ isPreview }}'
            equals: "true"
      - field: spec.environment
        default: production
        when:
          - field: '{{ isPrimary }}'
            equals: "true"
```

**At reconcile time** (operatorBox resource templates) — the annotations survive the SSA patch and every subsequent resync. The operator reads them on every reconcile and routes child resource creation accordingly:

```yaml
onReconcile:
  custom:
    - apiVersion: argoproj.io/v1alpha1
      kind: Application
      metadata:
        name: "{{ .metadata.name }}"
        namespace: argocd
      spec:
        syncPolicy:
          automated:
            prune: true
            selfHeal: true
      when:
        - field: '{{ isPrimary }}'
          equals: "true"

    - apiVersion: argoproj.io/v1alpha1
      kind: Application
      metadata:
        name: "{{ .metadata.name }}-preview"
        namespace: argocd
      spec:
        destination:
          namespace: "{{ .spec.targetNamespace }}-preview"
      when:
        - field: '{{ isPreview }}'
          equals: "true"
```

The CR looks the same in both cases. Admission gates what goes in; the operator routes what comes out. No second CRD, no feature flag, no if-else in a webhook — the surface that delivered the intent carries its name forward through the entire lifecycle.

---

## Response shaping per alias

Each alias can configure its own response — what the GET endpoint returns and what the apply response includes. This is independent of the CRD-level response config.

```yaml
serve:
  target:
    apifixture:
      primary: true
    preview:
      config:
        response:
          default: false            # no raw CR in response — payload fields only
          payload:
            phase: '{{ .status.phase }}'
            alias: '{{ getServeAlias . }}'
            workloadType: '{{ .spec.workloadType }}'
            environment: '{{ .spec.environment }}'
    internal:
      config:
        response:
          default: true             # full CR included
          payload:
            alias: '{{ getServeAlias . }}'
            target: '{{ getServeTarget . }}'
            source: '{{ getServeSource . }}'
```

`preview` callers get a restricted view — just the fields they need to act, nothing more. `internal` callers get the full CR plus provenance. The underlying resource is identical.

---

## Include files

Alias entries support `include:` — the same pattern as `serve.include`, `status.include`, and all other include fields. An included file may contain `tokens:` and `config:` at the top level. Inline fields take precedence.

The path is relative to the katalog file. Convention: `./serve/aliases/<name>.yaml`.

```yaml
serve:
  target:
    apifixture:
      primary: true
    preview:
      include: ./serve/aliases/preview.yaml
    internal:
      include: ./serve/aliases/internal.yaml
```

```yaml
# ./serve/aliases/preview.yaml
tokens:
  control-center:
    permissions:
      global: [get, list]
config:
  response:
    default: false
    payload:
      phase: '{{ .status.phase }}'
      alias: '{{ getServeAlias . }}'
```

---

## Disabling an alias or primary surface

Any entry in `serve.target` can be disabled independently:

```yaml
serve:
  target:
    apifixture:
      primary: true
      enabled: false    # primary surface closed; config authority still active
    internal:
      include: ./serve/aliases/internal.yaml
```

Disabled entries return the same response as unknown targets — no signal to callers. The primary's `enabled: false` closes the generic entry point while keeping the primary's token and response config as CRD-level defaults for other entries. This is the migration path when all callers have moved to named aliases.

`ork validate` warns when the primary is disabled and no enabled aliases remain — serve is configured but unreachable.

---

## Immutable routing surface

A resource's routing surface — the target or alias it was created through — is **immutable without explicit intent**.

Once a CR is applied via `preview`, only `preview` can update it. Attempting to update the same resource through `apifixture` (or any other surface) is rejected:

```json
{
  "accepted": false,
  "message": "routing surface conflict: resource was created via \"preview\", cannot update via \"apifixture\" without ?override=true"
}
```

To intentionally change the surface, pass `?override=true`. The gateway re-stamps the annotation with the new surface and logs a warning with the before/after values.

This is a documented safety property. A CI pipeline cannot accidentally re-route a preview resource to the production target. A developer cannot escalate their own resource to a surface they hold a token for. The surface that delivered the intent is the surface that owns the resource — until the operator explicitly changes it.

| Scenario | Result |
|---|---|
| Same surface → same surface | Allowed |
| Any surface → different surface | Rejected unless `?override=true` |
| New CR (no annotation) | Allowed on any surface |
| CR created by direct `kubectl apply` | No annotation — allowed on any surface |

---

## Annotation-driven response shaping on GET

When a CR is fetched via `GET /api/v1/resources/{kind}/{namespace}/{name}`, the gateway reads the `serve-alias` annotation stamped at apply time and uses it to select the alias-specific response config automatically.

The caller does not need to pass `?target=preview`. The CR carries its own surface.

Resolution order:

1. Alias from the URL path (if the caller addressed the resource via an alias route)
2. `orkestra.orkspace.io/serve-alias` annotation on the stored CR
3. CRD-level config (no alias)

CRs that bypassed the Gateway (direct `kubectl apply`) carry no annotation. CRD-level config applies and `isDirectApply` returns `true` in note expressions.

---

## What this enables

Aliases and provenance together give the operator a complete picture of *how* an intent arrived, not just *what* it said. A CR applied through `preview` carries that fact permanently. The operator can create a smaller, ephemeral environment. A CI system checking the `preview` alias gets back only the fields it needs. The platform team using `internal` gets the full CR and all provenance fields.

The same runtime. The same CRD. The same Kubernetes resource. Different surfaces, different contexts, different consequences — all without new CRDs, new webhooks, or new controllers.

---

## Where to go next

- [Schema reference — serve.target](../../reference/schema/02-katalog/20-serve.md#servetarget)
- [Token scoping](03-token-scoping.md)
- [Target mode](02-target-mode.md)

**Inspect from the CLI:**
`ork serve targets` — list targets with alias count · `ork serve tokens --alias <name>` — effective token map for an alias · `ork serve response --alias <name>` — effective response config · `ork serve can-i --alias <name>` — permission check per alias

→ [CLI reference — ork serve](../../reference/cli/13-serve.md)
