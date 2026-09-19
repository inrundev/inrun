# Webhook credential verification

A `gateway.webhooks` entry doesn't authenticate the way a `gateway.api.auth.tokens` caller does. There's no `Authorization: Bearer` header to check — GitHub, GitLab, Slack, and generic JSON callers each sign or stamp their own outbound request differently, and the gateway verifies each on its own terms rather than forcing them into one shared scheme.

---

## Four schemes, one shape

| Source | Header | Mechanism | Reference |
|--------|--------|-----------|-----------|
| GitHub | `X-Hub-Signature-256` | HMAC-SHA256 over the raw request body, `sha256=<hex>` | [GitHub docs](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries) |
| GitLab | `X-Gitlab-Token` | Static shared-secret string comparison | [GitLab docs](https://docs.gitlab.com/user/project/integrations/webhooks/#validate-payloads-by-using-a-secret-token) |
| Slack | `X-Slack-Signature` | HMAC-SHA256 over `v0:{timestamp}:{body}`, ±5 minute replay window | [Slack docs](https://api.slack.com/authentication/verifying-requests-from-slack) |
| Generic | `X-Signature-256` | HMAC-SHA256 over the raw body — GitHub's own convention, reused so callers don't learn a second scheme | — |

Every comparison — signature bytes or the GitLab static token — runs through a constant-time comparison. A timing side channel on a byte-by-byte string equality check would let an attacker recover the secret one byte at a time; the gateway never takes that risk, even for a value platform-internal enough to feel low-stakes.

None of these are OIDC. OIDC requires the caller to actively fetch and present a signed JWT — that's the right model for a CI job calling `ork serve apply` with its own workflow identity (see `githubOIDC`/`gitlabOIDC` in [Serve token permissions](08-serve-permissions.md)), but it isn't how GitHub, GitLab, or Slack's own webhook *delivery* systems work. They sign a body or send a static token, full stop — there's no OIDC alternative to offer even if the gateway wanted one.

---

## Two credentials, not one, for GitHub and GitLab

A GitHub or GitLab entry needs `secretRef` to verify the *delivery* is genuine, and a separate `contentTokenRef` to *read* the file that changed:

```yaml
gateway:
  admission:
    webhooks:
      github:
        - name: payments-repo
          enabled: true
          path: /webhooks/github/payments
          branch: main
          watch:
            - "services/*/intent.yaml"
          secretRef:
            name: ork-payments-github-secret
            key: secret
          contentTokenRef:
            name: ork-payments-github-app-token
            key: token
```

A push event's payload carries only the *paths* that changed — GitHub and GitLab never include file content in the webhook body itself. Fetching what changed means a second, separate API call (the Contents API for GitHub, the Repository Files API for GitLab), and that call needs its own credential — a GitHub App installation token or a GitLab API token with read access to the repo.

This split matters for blast radius: `secretRef` only proves a request came from GitHub/GitLab's webhook infrastructure — it grants no read access on its own. A leaked `secretRef` lets an attacker forge webhook *deliveries*, not read repository content. `contentTokenRef` is the credential that can actually see inside the repo, and it's never used for delivery verification.

`contentTokenRef` is also the one exception to the self-bootstrap behavior below: it's minted externally (by GitHub or GitLab, not by Orkestra), so the gateway only reads and rotates it — it can never generate one from nothing the way it can a shared secret.

!!! note "Public GitHub/GitLab only"
    Content fetches go to `api.github.com` and `gitlab.com/api/v4` directly. GitHub Enterprise Server hosts its API at a different base path (`https://{host}/api/v3/...`), and self-hosted GitLab instances aren't reachable either — neither is handled today.

---

## Self-bootstrap and rotation, for free

`secretRef`, `contentTokenRef`, and `signingSecretRef` are all the same `APISecretRef` shape `gateway.api.auth.tokens[].secretRef` already uses:

```yaml
secretRef:
  name: ork-payments-github-secret
  key: secret
  rotateAfter: 90d
```

That reuse isn't cosmetic. Every webhook credential gets the same behavior a `gateway.api.auth.tokens` secret gets, with no extra code path:

- **Self-bootstrap** — if the named Secret doesn't exist on first startup, the gateway generates a UUIDv4 value and creates it. There's no manual step to wire up a `secretRef`-based entry from a cold start; read the generated value back out and hand it to GitHub/GitLab/Slack's webhook settings.
- **Rotation** — `rotateAfter` (e.g. `90d`) is checked against an age annotation on the Secret at every token reload cycle. Past the window, the gateway deletes and recreates it with a fresh value — the entry picks it up automatically, no restart.

`contentTokenRef` is read-only in this flow — self-bootstrap never applies to it, since a self-generated random string wouldn't be a valid GitHub/GitLab API token in the first place.

---

## `serve.tokens` treats a webhook entry exactly like a bearer token

An entry's own `name` field does double duty: it's the identity `serve.tokens` authorizes against, *and* the value stamped as the `serve-source` provenance annotation on every CR it applies — the same slot an OIDC caller's verified `sub` claim fills.

```yaml
serve:
  tokens:
    payments-repo:          # the github webhook entry's own Name
      permissions:
        global: ["create", "update"]
```

`ork validate` enforces that every `serve.tokens` key resolves to something real — either a `gateway.api.auth.tokens` entry or a `gateway.webhooks` entry's `name` — the same "no typo'd token name" guarantee [Serve token permissions](08-serve-permissions.md) describes for bearer tokens. Nothing about the permission model changes because the caller is a webhook instead of a bearer token: no restriction declared means no restriction enforced; once any restriction is declared, an un-listed name is denied.

---

## Delivery failures are reported, never a delivery-protocol failure

GitHub and GitLab retry a webhook delivery that returns a non-2xx response. A downstream apply rejection — a denied token, a failed admission rule, a missing required field — is not a delivery failure worth retrying, so every intake handler acks `200` once signature/token verification passes, and reports per-outcome results in the response body instead of the HTTP status:

```json
{"applied": [{"path": "services/payments/intent.yaml", "accepted": false, "message": "team is required"}]}
```

Only signature/token verification failures (`401`) and malformed requests (`400`/`405`) use non-2xx status codes — the layer GitHub/GitLab's retry logic is actually meant to catch.

---

## Testing verification locally

`ork webhook play` runs the branch filter, watch-pattern match, and apply chain locally — but deliberately skips signature/token verification, since there's no real secret exchange to test outside a live deployment. See [CLI reference — ork webhook](../reference/cli/16-webhook.md) and [Webhook Intake](../concepts/self-service/09-webhook-intake.md) for the full local-testing story.

---

## Where to go next

- **[Serve token permissions](08-serve-permissions.md)** — the authorization layer every authenticated caller, webhook or bearer token, is checked against
- **[Webhook Intake](../concepts/self-service/09-webhook-intake.md)** — the concept this verification model supports
- **[CLI reference — ork webhook](../reference/cli/16-webhook.md)** — `ork webhook list` / `ork webhook play`
