# API Evolution

Upgrading a Motif changes the implementation. Evolving the CRD API changes the schema — what fields users write in their CRs. These are separate concerns, and Orkestra handles both.

This page covers field restructuring: moving `spec.port` (flat, v1) into `spec.expose.port` (structured, v2). Two strategies: with a conversion webhook, without one.

---

## The scenario

```yaml
# v1 CR — flat, terse
spec:
  image: ghcr.io/orkspace/orkestra-dev-server:0.7.5
  port: 9999

# v2 CR — structured, self-documenting
spec:
  image: ghcr.io/orkspace/orkestra-dev-server:0.7.5
  expose:
    port: "9999"
    host: api.example.com
    protocol: HTTPS    # new in v2 — no v1 equivalent
```

The question is: what happens to the clients still writing v1 CRs?

---

## with-webhooks: bidirectional conversion via Orkestra Gateway

Use when **external clients are locked to v1** — CI pipelines, Terraform, other teams that own CRs you cannot update. Both API versions must stay alive simultaneously.

The API server stores objects in v2 (the current version). When a client writes a v1 CR, Orkestra converts it to v2 before storage. When a client reads as v1, Orkestra converts it back from v2 on the way out.

The conversion paths are four template lines in the Katalog:

```yaml
security:
  admission:
    conversion:
      enabled: true
spec:
  crds:
    webapp-v2:
      admission:
        conversion:
          storageVersion: v1
          updateCRD: true
          paths:
            # v1 → v2: lift flat fields into expose struct
            - from: v1
              to: v2
              spec:
                expose:
                  port: "{{ .spec.port }}"
                  host: '{{ default "" .spec.host }}'
                  protocol: HTTP

            # v2 → v1: flatten expose back to top-level fields
            - from: v2
              to: v1
              spec:
                port: "{{ .spec.expose.port }}"
                host: '{{ default "" .spec.expose.host }}'
```

Conversion runs in-process inside Orkestra Gateway — no separate webhook pod, no TLS management, no cert-manager. The `/convert` endpoint is the same process that runs the operator.

**Deploy with Gateway enabled:**

```bash
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --set gateway.enabled=true \
  --wait --timeout 120s
```

**Verify the round-trip:**

```bash
# v1 CR read back as v1 — flat port reconstructed
kubectl get webapps.v1.rkguide.demo my-webapp -o yaml | grep port
# port: 9999

# Same object read as v2 — expose block present
kubectl get webapps.v2.rkguide.demo my-webapp -o yaml | grep -A4 expose:
# expose:
#   port: "9999"
#   protocol: HTTP
```

**Observe conversions:**

```bash
# In a separate terminal:
ork proxy --for gateway
```

```bash
curl localhost:8443/katalog/webapp-v2 | jq '.conversion'
# {"enabled":true,"total":4,"failures":0,"avgLatencyMs":0.62}
```

---

## without-webhooks: normalize collapses both formats

Use when **your team controls all CRs** — CI/CD writes them, no external client. You want to migrate gradually: existing CRs that use `spec.port` keep working while you update them one by one.

No multi-version CRD. No webhook. No TLS. One version, one CRD, one normalize block.

```yaml
normalize:
  spec:
    # v2: spec.expose present → use expose.port directly
    # v1: spec.expose absent  → lift spec.port
    expose.port: '{{ if .spec.expose }}{{ .spec.expose.port }}{{ else }}{{ .spec.port }}{{ end }}'
    # v2: spec.expose present → use expose.host directly
    # v1: spec.expose absent  → lift spec.host
    expose.host: '{{ if .spec.expose }}{{ .spec.expose.host | default "" }}{{ else }}{{ .spec.host | default "" }}{{ end }}'
```

Before any mutation, validation, or template rendering:
- If the CR has `spec.expose.port` → use it.
- If the CR has `spec.port` → use that (converted to string).
- Downstream logic always sees `spec.expose.port` — no branching anywhere.

**Both CRs reconcile identically:**

```bash
kubectl apply -f cr-port-string.yaml      # old format: spec.port: 9999
kubectl apply -f cr-port-structured.yaml  # new format: spec.expose.port: "9999"

kubectl get webapps
# NAME               PORT   FORMAT             PHASE
# webapp-flat        9999   flat (deprecated)  Running
# webapp-structured  9999   structured         Running
```

**Migrating to pure v2:**

Once all CRs are updated to `spec.expose`:
1. Remove the `normalize` block.
2. Add a validation rule: `field: spec.port, operator: notExists, action: deny`.
3. Re-run `ork simulate` + `ork e2e`.
4. Publish v1.3.0.

No webhook to remove. No cert to rotate. No stored object migration.

---

## Decision guide

| | with-webhooks | without-webhooks |
|---|---|---|
| CRD versions | v1 + v2 (multi-version) | v1 only (single version) |
| Conversion webhook | Orkestra Gateway `/convert` | Not needed |
| External clients locked to v1 | Supported — bidirectional conversion | Not supported |
| Gateway required | Yes (`--set gateway.enabled=true`) | No |
| Best for | Public APIs, external consumers | Internal operators, platform teams |

---

## Try it

```bash
ork init --pack registry-guide
cd 07-upgrade/api-evolution

# Follow the steps in the README
```

→ Next: [Policy](09-policy.md)
