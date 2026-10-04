# pkg/gateway/api

`api` implements the Orkestra Gateway API — a REST surface served by the gateway process for creating, reading, deleting, and discovering the schema of Custom Resources managed by Orkestra, without a kubeconfig or kubectl. Callers submit flat fields (**target mode**); the gateway builds and applies the Kubernetes CR. Full CR submission still works for advanced callers, but it isn't the primary contract.

```
POST   /api/v1/apply                              ← {"target": "...", ...fields} or a full CR
GET    /api/v1/resources/{kind}/{namespace}/{name}
GET    /api/v1/resources/{kind}/{namespace}
DELETE /api/v1/resources/{kind}/{namespace}/{name}
GET    /api/v1/schema                             ← catalog — every target this caller can see
GET    /api/v1/schema?target=<t>                  ← flat field contract for one target
GET    /api/v1/raw-schema?kind=<k>                ← raw OpenAPI v3 schema, for advanced callers
```

## Enabling

The Gateway API is off by default. Enable it in the Katalog:

```yaml
gateway:
  api:
    enabled: true
    auth:
      tokens:
        - name: ci-pipeline
          token: "${CI_ORK_TOKEN}"
```

Enable the Create button in the Control Center per CRD, and give it a caller-facing identifier:

```yaml
spec:
  crds:
    application:
      serve:
        enabled: true
        target: smartapp
        fields:
          repository:
            label: "Repository"
            required: true
```

## What callers can do

| Caller | Path |
|--------|------|
| Control Center | `GET ?target=` to render a form, `POST` to submit, `GET /resources/...` to poll status |
| CI pipeline | `curl -X POST /api/v1/apply -d '{"target": "...", ...}'` — no kubeconfig in the runner |
| Terraform provider | create → `POST`, read → `GET`, destroy → `DELETE` |
| Slack bot | parse command → `POST` flat fields → poll `pollUrl` → reply |
| Any HTTP client | any language, any platform — none of them need to know what a CRD is |

Security — admission rules, namespace protection, deletion protection — is enforced exactly as it is for `kubectl apply`, through the gateway webhook path, for both target mode and full CR mode. `serve.tokens` adds one thing specific to this API: per-token, per-CRD operation and namespace scoping — see [05-auth.md](docs/05-auth.md#scoping--servetokens).

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand why the Gateway API lives in the gateway | [docs/01-overview.md](docs/01-overview.md) |
| Understand the full apply pipeline (auth → validation → admission → SSA) | [docs/02-apply.md](docs/02-apply.md) |
| Understand the schema endpoint and how CRD schemas are resolved | [docs/03-schema.md](docs/03-schema.md) |
| Understand GET and DELETE, deletion protection, polling patterns | [docs/04-resources.md](docs/04-resources.md) |
| Understand token auth, add a new auth mode | [docs/05-auth.md](docs/05-auth.md) |
