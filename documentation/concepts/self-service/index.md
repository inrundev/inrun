# Self-Service

Every Orkestra operator is self-service by default. The same gateway that the platform team uses to manage CRDs is the API every caller — a browser form, a CI pipeline, a Slack bot, a CLI — uses to interact with them. No separate portal. No secondary system to wire up.

The serve layer is what makes it work. It sits between callers and the operator, translating intent into Kubernetes objects without the caller knowing what an `apiVersion` is.

---

## The model

A platform team defines a CRD and a Katalog. That Katalog already declares what the operator does — what children it creates, what validation rules apply, what the status looks like. Adding `serve:` to the CRD entry turns the same Katalog into a self-service API:

```yaml
spec:
  crds:
    application:
      serve:
        enabled: true
        target: app
        namespace: '{{ teamName }}'
        fields:
          image:
            label: "Container Image"
            required: true
          environment:
            label: "Environment"
            type: enum
            enum: ["staging", "production"]
```

That is the whole declaration. One addition to an existing file. What it produces:

- `GET /api/v1/schema` — lists `app` as a self-service target
- `GET /api/v1/schema?target=app` — returns the flat field contract
- `POST /api/v1/apply` — accepts `{"target":"app","name":"...","image":"...","environment":"staging"}` and creates the CR
- `GET /api/v1/resources/app/...` — reads status back
- The Control Center shows a `[+ Create]` button and a generated form

Every caller hits the same API. Every request goes through the same validation rules and token checks before anything reaches etcd.

---

## What the serve layer does

A caller submits flat fields. The serve layer:

1. Resolves the target to a CRD
2. Checks the caller's token and permissions
3. Builds the full CR from the submitted fields (`spec`, `metadata.name`, `metadata.namespace`, labels, annotations)
4. Stamps provenance annotations
5. Evaluates `validation.rules`
6. Applies via server-side apply
7. Returns the response (default: the built CR, or a shaped payload if `serve.config.response` is declared)

The caller never sees the CRD schema, the Kubernetes object structure, or any of the operator internals. They see what they submitted and what the status says.

---

## Field translation

The serve layer can transform submitted fields before writing to the CRD. Callers speak one vocabulary; the CRD speaks another. `serve.fields.value` and `serve.fields.values` bridge them:

```yaml
fields:
  schedule:
    label: "Schedule (cron)"
    required: true
    values:
      schedule.minute:     '{{ cronMinute .value }}'
      schedule.hour:       '{{ cronHour   .value }}'
      schedule.dayOfMonth: '{{ cronDom    .value }}'
      schedule.month:      '{{ cronMonth  .value }}'
      schedule.dayOfWeek:  '{{ cronDow    .value }}'
```

Caller submits `"0 2 * * 1-5"`. CRD receives a structured schedule object. Neither side sees the other's format.

→ [Field Translation](08-field-translation.md)

---

## Token scoping

Not every caller gets the same access. `serve.tokens` restricts which gateway tokens can reach a CRD, with per-token operation and namespace permissions:

```yaml
serve:
  tokens:
    ci-pipeline:
      namespaces: ["team-payments-staging"]
      permissions:
        resources: ["create", "update"]
```

→ [Token Scoping](03-token-scoping.md)

---

## Multiple surfaces, one CRD

A CRD can expose multiple named targets — aliases — alongside the primary. A `preview` alias creates lightweight environments; an `internal` alias returns a richer response. The same CR, the same operator, different surfaces.

→ [Aliases and Intent Provenance](04-aliases-and-provenance.md)

---

## Webhook intake

`ork serve apply` is pull-based — someone runs a command. `gateway.webhooks` is push-based — GitHub, GitLab, Slack, or a generic JSON caller triggers the apply on its own, with no CLI invocation in the loop. Every source ends at the same target-mode chain everything else in this document describes; the source's only job is turning its own payload into a flat field map.

```yaml
gateway:
  admission:
    webhooks:
      slack:
        - name: platform-workspace
          path: /webhooks/slack
          signingSecretRef: { name: ork-slack-signing-secret, key: secret }
          commands: ["/deploy"]
```

→ [Webhook Intake](09-webhook-intake.md)

---

## Testing without a cluster

```bash
ork serve validate          # check serve config
ork serve validate --full   # show target, fields, token map
ork serve play -f katalog.yaml --token dev -i intent.yaml   # run the full chain locally
ork webhook play -f katalog.yaml --source slack --webhook platform-workspace \
  --command /deploy --text "app name=foo team=bar"          # same chain, from a webhook payload
```

→ [Local Intent Testing](06-local-intent-testing.md) · [Webhook Intake](09-webhook-intake.md)

---

## Use cases

The serve layer is general-purpose. The same mechanism works for:

- Internal Developer Platforms — developer self-service for infrastructure and application CRDs
- CI/CD pipelines submitting deploy intents via token
- Slack bots posting to the gateway on user request
- Any caller that needs to create or update Kubernetes resources without knowing the CRD schema

---

## Where to go next

- [Target Mode](02-target-mode.md) — how flat fields become a CR
- [Additional Fields](01-labels-and-annotations.md) — labels, annotations, field hints
- [Field Translation](08-field-translation.md) — `value`, `values`, intent gating
- [Token Scoping](03-token-scoping.md)
- [Aliases and Intent Provenance](04-aliases-and-provenance.md)
- [Local Intent Testing](06-local-intent-testing.md)
- [Live Delivery](07-live-delivery.md)
- [Webhook Intake](09-webhook-intake.md)

- [Gateway API reference](../../reference/schema/02-katalog/17-gateway-api.md)
- [Webhook credential verification](../../security/09-webhook-verification.md)
