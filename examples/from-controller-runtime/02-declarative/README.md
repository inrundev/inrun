# 01 — Declarative

The same WebApp operator. Zero Go. No build step. No image.

`ork run` reads the Katalog, installs the CRD, starts the informer, wires the workqueue, manages leader election, and reconciles every `WebApp` CR — creating the Deployment and Service from the templates. The behaviour is identical to the controller-runtime baseline.

---

> **Before you start:** Update [katalog.yaml](katalog.yaml) — set `author:` to your name or org and `metadata.name` to your operator name. Set your registry before pushing:
> ```bash
> export ORK_REGISTRY=ghcr.io/myorg/katalogs
> ```

---

## What changed

| | controller-runtime | Declarative Katalog |
|---|---|---|
| Lines of Go | ~150 | 0 |
| Reconcile loop | hand-written | runtime |
| Deployment spec | hard-coded in controller | `operatorBox.onCreate.deployments` |
| Service spec | hard-coded in controller | `operatorBox.onCreate.services` |
| Owner references | `metav1.NewControllerRef(...)` | automatic |
| Idempotency | `Get` → `IsNotFound` → `Create/Update` | automatic |
| Status propagation | `Status().Update()` | `status.fields` |
| Drift correction | not implemented | `reconcile: true` |
| `ork simulate` | not implemented | behavioral proof in-memory against a fake cluster|
| `ork e2e` | not implemented | behavioral proof against a real cluster |
| Build & deploy | Dockerfile, image push, Helm | `ork push` and `ork run` |
| Supply chain integrity | not implemented | `ork inspect` |

---

## Step 1 — Simulate

Prove the Deployment and Service are created — no cluster, no runtime:

```bash
ork simulate
```

Expected:

```
Cycle 1:
  + deployments/my-webapp
  + services/my-webapp-svc
  ~ status/my-webapp
✓ create deployments/my-webapp (cycle 1)
✓ create services/my-webapp-svc (cycle 1)
PASS
```

---

## Step 2 — Run locally

Validate first:

```bash
ork validate
```

Run:

```bash
ork run
```

No cluster? Add `--dev` to create a temporary kind cluster:

```bash
ork run --dev
```

Apply the CR and verify:

```bash
kubectl get webapps
kubectl get deployments
kubectl get services
kubectl get webapp my-webapp -o yaml | grep -A5 "status:"
```

---

## Step 3 — Control center

In a second terminal:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) to see live operator state — CRD health, worker activity, reconcile metrics, and the `WebApp` CR you applied.

---

## Step 4 — Push to the registry

Update the `author:` field in [katalog.yaml](katalog.yaml) to your name or org before publishing, then set your registry and push:

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

`ork push` runs simulate and e2e as gates before publishing. The artifact is only written to the registry if both pass. Gate results are stored as OCI annotations — visible to any consumer via `ork inspect`.

---

## Step 5 — Inspect

```bash
ork inspect webapp-declarative:1.0.0
```

---

## Adding another operator

A second CRD is another entry in `spec.crds:`. This example includes a `worker` CRD alongside `webapp`:

- [`crd-with-secret.yaml`](crd-with-secret.yaml) — the `Worker` CRD
- [`cr-with-secret.yaml`](cr-with-secret.yaml) — a sample `Worker` CR

The worker entry generates a token secret with automatic rotation every 30 days (`rotateAfter: 30d`) and injects it into the worker Deployment. No new project, no new binary:

```yaml
worker:
  crdFile: ./crd-with-secret.yaml
  operatorBox:
    reconcile:
      onCreate:
        secrets:
          - name: "{{ .metadata.name }}-token"
            once: true
            rotateAfter: 30d
            data:
              token: "{{ randomAlphanumeric 32 }}"
        deployments:
          - name: "{{ .metadata.name }}"
            image: "{{ .spec.image }}"
            replicas: "{{ .spec.replicas }}"
            reconcile: true
            env:
              - name: WORKER_TOKEN
                valueFrom:
                  secretKeyRef:
                    name: "{{ .metadata.name }}-token"
                    key: token
```

Uncomment and run:

```bash
ork run
```

Apply the second CR:

```bash
kubectl get workers
kubectl get deployments
kubectl get secrets
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[02 — hooks](../02-hybrid/README.md) — keep the Deployment declarative, write the Service in Go.
