# Profiles 08 — LimitRange

One CR. Three LimitRanges. Each uses a profile declared in the Katalog's `profiles:` block — LimitRange has no built-in presets, so the `profiles:` block is the only source.

**What you learn:** `limitRanges.profile`, how to declare user-defined LimitRange profiles, and that `ork validate` enforces references against your own registry.

---

## Profiles at a glance

| Profile | Default CPU | Default memory | Max CPU | Max memory |
|---|---|---|---|---|
| `minimal` | 200m | 128Mi | 1 | 512Mi |
| `standard` | 500m | 512Mi | 2 | 4Gi |
| `generous` | 1 | 2Gi | 4 | 8Gi |

All three set both `default` (applied when a container omits `resources.limits`) and `defaultRequest` (applied when a container omits `resources.requests`).

---

## Step 1 — Validate

```bash
ork validate
```

## Step 2 — Simulate

```bash
ork simulate
```

---

## Step 3 — Start the runtime

```bash
ork run
```

---

## Step 4 — Apply the CR

In a separate terminal:

```bash
kubectl apply -f ../cr.yaml
```

Verify the limit ranges and inspect an expanded profile:

```bash
kubectl get limitrange
kubectl describe limitrange my-service-limits-standard
```

---

## Using a profile in your own Katalog

Declare the profile in the `profiles:` block, then reference it from `operatorBox`:

```yaml
profiles:
  limitRanges:
    - name: standard
      limits:
        - type: Container
          default: { cpu: 500m, memory: 512Mi }
          defaultRequest: { cpu: 100m, memory: 128Mi }

spec:
  crds:
    mycrd:
      operatorBox:
        reconcile:
          onCreate:
            limitRanges:
              - name: "{{ .metadata.name }}-limits"
                namespace: "{{ .metadata.namespace }}"
                profile: standard
                reconcile: true
```

---

## E2E

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
