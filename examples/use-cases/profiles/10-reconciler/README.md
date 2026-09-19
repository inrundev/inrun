# Profiles 10 — Reconciler

One CR. One Deployment. The reconciler's workers, resync interval, and queue depth come from a profile name — no inline values on the CRD entry.

**What you learn:** how `reconcile.profile` references a named preset; the difference between built-in profiles and user-defined ones; that the profile expands at `ork validate` time and any inline field on `operatorBox.reconcile` overrides the profile.

**Built-in profiles** (no declaration needed):

| Profile | workers | resync | queue.maxDepth | Use for |
|---|---|---|---|---|
| `high-throughput` | 10 | 5m | 1000 | High-volume operators, many CRs |
| `conservative` | 2 | 1m | 100 | Low-change production operators |
| `development` | 1 | 30s | 50 | Local dev, predictable log ordering |

**User-defined profiles** (declared in this Katalog):

| Profile | workers | resync | queue.maxDepth | Use for |
|---|---|---|---|---|
| `api-service` | 4 | 30s | 200 | Balanced web service operator |
| `batch-worker` | 2 | 5m | 500 | Background processing operator |
| `local-dev` | 1 | 15s | 50 | Iterative local development |

---

## Step 1 — Start with a built-in profile

Open [`katalog.yaml`](katalog.yaml) and set the reconciler to use the `conservative` built-in:

```yaml
operatorBox:
  reconcile:
    profile: conservative
```

Validate — no `profiles:` block needed for built-ins:

```bash
ork validate
```

You should see `workers: 2 / resync: 1m` in the output. The built-in expanded without any declaration.

---

## Step 2 — Simulate

```bash
ork simulate
```

---

## Step 3 — Run and inspect

```bash
ork run
```

In a separate terminal, apply the CR and check the reconciler config:

```bash
kubectl apply -f ../cr.yaml
curl -sf http://localhost:8080/katalog/service | jq '{workers,resync,maxDepth,workersSource,resyncSource}'
```

Expected with `conservative`:

```json
{
  "workers": 2,
  "resync": "1m0s",
  "maxDepth": 100,
  "workersSource": "configured",
  "resyncSource": "configured"
}
```

---

## Step 4 — Switch to a user-defined profile

Stop the runtime (`Ctrl+C`). Edit [`katalog.yaml`](katalog.yaml) to use the user-defined `local-dev` profile instead:

```yaml
operatorBox:
  reconcile:
    profile: local-dev
```

The `local-dev` profile is declared in the `profiles:` block at the top of `katalog.yaml`. Validate again:

```bash
ork validate
```

Output now shows `workers: 1 / resync: 15s`. Re-run:

```bash
ork run
```

Check the endpoint again:

```bash
curl -sf http://localhost:8080/katalog/service | jq '{workers,resync,maxDepth,workersSource,resyncSource}'
```

Expected:

```json
{
  "workers": 1,
  "resync": "15s",
  "maxDepth": 50,
  "workersSource": "configured",
  "resyncSource": "configured"
}
```

The only change was one line in `katalog.yaml`. No code. No rebuild.

---

## Using reconciler profiles in your own Katalog

```yaml
profiles:
  reconcile:
    - name: api-service
      workers: 4
      resync: 30s
      queue:
        maxDepth: 200
spec:
  crds:
    mycrd:
      operatorBox:
        reconcile:
          profile: api-service
          # inline fields override the profile — add here to tune per-CRD
```

To use a built-in without declaring it:

```yaml
operatorBox:
  reconcile:
    profile: high-throughput
```

---

## E2E

This asserts that the `local-dev` profile expands correctly.

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
