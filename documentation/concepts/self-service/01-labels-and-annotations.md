# Labels and Annotations as Form Fields

`serve.fields` exposes `spec.*` — but not everything a form should collect belongs in `spec`. A team name, an environment, a feature flag are metadata concepts in Kubernetes: labels and annotations, not workload data the operator computes with. `serve.labels` and `serve.annotations` close that gap — they expose label and annotation keys as self-service form fields, written to `metadata.labels`/`metadata.annotations` on apply instead of `spec`.

```yaml
spec:
  crds:
    application:
      serve:
        enabled: true
        fields:
          image:
            label: "Container Image"
            order: 1
        labels:
          team:
            label: "Team"
            placeholder: "team-payments"
            order: 2
        annotations:
          canary.myorg.io:
            label: "Enable canary rollout"
            type: boolean
            order: 3
```

Callers submit all three as flat fields — no bucketing, no Kubernetes structure. The gateway routes each one to the right destination at apply time: `image` → `spec.image`, `team` → `metadata.labels["team"]`, `canary.myorg.io` → `metadata.annotations["canary.myorg.io"]`.

---

## Why this matters beyond convenience

Once a value lives in `metadata.annotations`, every template expression Orkestra already supports can read it back — `when:` conditions, `value:` resolution, `normalize:`, profile selection — via the `getLabel`/`getAnnotation`/`hasAnnotation`/`hasLabel` notes. That makes annotations a **schema-free configuration channel**: a platform team can add a new decision point — a rollout gate, a tier, a feature flag — without a CRD schema change and without a code change.

This is the same idiom `nginx-ingress-controller` and `cert-manager` already use (`nginx.ingress.kubernetes.io/rewrite-target`, `cert-manager.io/cluster-issuer`) — annotations as the extension point every serious controller reaches for once a field stops being "data the workload needs" and becomes "a knob an operator reads." `serve.annotations` is what makes that annotation settable through a form, instead of requiring a hand-edited CR.

```yaml
spec:
  crds:
    application:
      serve:
        enabled: true
        annotations:
          canary.myorg.io:
            label: "Enable canary rollout"
            type: boolean

      operatorBox:
        reconcile:
          onReconcile:
            deployments:
              - name: "{{ .metadata.name }}"
                replicas: '{{ ternary (eq (getAnnotation . "canary.myorg.io") "true") "1" "3" }}'
```

No new spec field, no CRD schema change — the platform team gets an annotation-driven extension point using notes that already ship.

---

## Choosing labels vs. annotations

The same question Kubernetes itself answers for every native resource:

- **Label** — if you would ever want to *select* on it (filter, group, build a dashboard by it). Team, environment, and tier are almost always labels in real clusters, for exactly this reason.
- **Annotation** — everything else: feature flags, external reference IDs (a Jira ticket, a Terraform run ID), free-form configuration. Not used for selection, not indexed.

Both go through the same `type`/`enum` presentation hints as `serve.fields` — the difference is only the write target, including `order`: it competes for the same sequence as `serve.fields`, not a separate one per bucket. A `serve.labels` entry and a `spec` field sharing a non-zero `order` is a load-time error the same as two label entries colliding with each other — see [why `order` is more than form layout](index.md#the-orkestra-model).

---

## Hand-written rules on labels or annotations need `link:`

`required`/`type: enum` violations on `serve.labels`/`serve.annotations` already highlight the right field on the form automatically — that's synthesized for you. A hand-written `validation.rules` entry on the same field doesn't, unless you tell it to: label and annotation entries resolve through a `getLabel`/`getAnnotation` template expression (or a `notes:` function built on one), not a plain `spec.*` path, so the violation reports that raw expression as the offending field — nothing a client can match back to what it rendered.

```yaml
serve:
  labels:
    team:
      label: "Team"
      required: true

admission:
  validation:
    rules:
      - field: '{{ isDNS1123Subdomain (getLabel . "team") }}'
        link: team
        equals: "true"
        message: "team must be a valid DNS subdomain"
        action: deny
```

`link: team` is what makes the Control Center highlight the Team field for this rule the same way it already does for the synthesized ones. It also means the check doesn't have to live in one giant expression — several focused rules can `link:` to the same field, each with its own message for its own failure mode.

→ [Linking a rule to its form field](../../reference/schema/02-katalog/07-validation.md#linking-a-rule-to-its-form-field-link)

---

## A gotcha with boolean flags

A checkbox field in the Control Center form always submits an explicit value — unchecked submits `"false"`, not an absent annotation. That means a value-based check (`getAnnotation . "canary.myorg.io" | equals "true"`) is correct for a form-driven boolean field; `hasAnnotation` is not — it treats *any* non-empty value, including the string `"false"`, as present. Reserve `hasAnnotation` for annotations that are genuinely presence-only (set by automation, never toggled through a checkbox).

---

<!-- ## Try It

First example — `01-form` drives its whole flow from `serve.labels`/`serve.annotations` and minimal `spec`
```bash
ork init --pack use-cases/idp

# Follow steps in the README
```
 -->
---

## Where to go next

→ [`serve.labels` / `serve.annotations` schema reference](../../reference/schema/02-katalog/20-serve.md)

→ [Kubernetes notes — getLabel, getAnnotation, hasAnnotation](../notes/index.md)

→ [Required fields are enforced automatically](../../reference/schema/02-katalog/07-validation.md#required-fields-are-enforced-automatically) — `required: true` synthesizes server-side enforcement, not just a form hint

→ [Enum fields are validated automatically](../../reference/schema/02-katalog/07-validation.md#enum-fields-are-validated-automatically) — `type: enum` synthesizes an `in` rule the same way

→ [Serve-aware messages for hand-written rules](../../reference/schema/02-katalog/07-validation.md#serve-aware-messages-for-hand-written-rules) — write `message:` using the field's `label:`, not its raw `spec.*` path

→ [Linking a rule to its form field (`link:`)](../../reference/schema/02-katalog/07-validation.md#linking-a-rule-to-its-form-field-link) — full field reference and the load-time checks `link:` gets
