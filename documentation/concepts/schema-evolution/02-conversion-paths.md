# conversion.paths:

`conversion.paths:` declares how Orkestra translates a CR between API versions. The Kubernetes API server calls Orkestra Gateway's `/convert` endpoint whenever a version mismatch exists — reading a v1 object stored as v2, or writing a v2 object that will be stored as v1.

No Go. No separate webhook deployment. No certificate to manage. The same Gateway that handles admission webhooks handles conversion.

---

## When to use it

Use `conversion.paths:` when you have committed to a multi-version CRD:

- External clients target a specific API version
- You have live objects at v1 that must remain readable as v1
- You want the API server to enforce the schema at admission time per version

If you just want to tolerate different input shapes in a single-version CRD, use [`normalize:`](./01-normalize.md) instead — no webhook, no TLS, no caBundle.

---

## The Katalog declaration

```yaml
spec:
  crds:
    cronjob-v2:
      apiTypes:
        group: demo.orkestra.io
        version: v2
        kind: CronJob
        plural: cronjobs
      admission:
        conversion:
          storageVersion: v1    # all objects stored as v1
          updateCRD: true       # Orkestra patches caBundle on startup

          paths:
            - from: v1
              to: v2
              spec:
                schedule: "{{ cronToMap .spec.schedule }}"

            - from: v2
              to: v1
              spec:
                schedule: "{{ cronFromMap .spec.schedule }}"
```

`updateCRD: true` tells Orkestra to patch the CRD's `spec.conversion.webhook.clientConfig.caBundle` at startup with the CA certificate it generated for Gateway. You do not base64-encode anything or touch the CRD spec manually.

Declare both directions. Every field that changes between versions must appear in both paths. Fields that do not change pass through unchanged with `"{{ .spec.fieldName }}"`.

---

## The caBundle is continuous, not one-shot

Orkestra does not patch the caBundle once and hope it stays. Two mechanisms keep it current:

**CRD watch** — a goroutine watches every conversion CRD for `MODIFIED` events. Any external change to the webhook config triggers an immediate re-patch. Detected within one API round-trip.

**Safety ticker** — a background timer re-patches every 30 seconds as a backstop, covering anything the watch missed (stream interruption, edge cases).

If someone runs a cleanup job that removes the caBundle annotation, Orkestra restores it within 30 seconds. The protection is continuous.

---

## What happens at conversion time

When a client requests a CR at v2 and the object is stored at v1:

1. API server calls `POST /convert` on Orkestra Gateway
2. Gateway decodes the `ConversionReview`, finds the `from: v1 → to: v2` path
3. Builds a resolver from the source object
4. Evaluates each field expression — `cronToMap "0 2 * * 1-5"` → `{minute:"0", hour:"2", ...}`
5. Returns the converted object with `apiVersion` updated to v2

The same note expression language used in reconcile templates works in conversion paths. No separate DSL to learn.

---

## The participant field

For the legacy version entry, set `conversion.participant: true` instead of re-declaring paths:

```yaml
cronjob-v1:
  apiTypes:
    group: demo.orkestra.io
    version: v1
    kind: CronJob
    plural: cronjobs
  admission:
    conversion:
      participant: true   # paths live on cronjob-v2; this entry joins the pair
```

`participant: true` tells the Control Center to show conversion stats for v1 without requiring you to duplicate path declarations. The v2 entry owns the paths; v1 participates.

---

## Observing conversions

```bash
ork proxy --for gateway

curl localhost:8443/katalog/cronjob-v2 | jq '.conversion'
```

```json
{
  "enabled": true,
  "total": 11279,
  "failures": 0,
  "avgLatencyMs": 0.59,
  "p95LatencyMs": 0.69
}
```

---

## Production results

The with-webhooks CronJob example has been run in production:

| | |
|---|---|
| v1 → v2 conversions | 5,024 |
| v2 → v1 conversions | 6,255 |
| Failures | **0** |
| v1 → v2 p95 latency | 0.69 ms |
| v2 → v1 p95 latency | 0.49 ms |

---

## Try it

```bash
ork init --pack use-cases/crd-conversion/with-webhooks
cd with-webhooks
# Follow the steps in README
```

This runs the multi-version CronJob operator. Apply a v1 CR, read it back as v2. Apply a v2 CR, read it back as v1. The schedule field converts losslessly in both directions.

---

## Where to go next

- **[normalize:](./01-normalize.md)** — single-version, no webhook, input tolerance
- **[Schema Evolution](./index.md)** — pick-one table
