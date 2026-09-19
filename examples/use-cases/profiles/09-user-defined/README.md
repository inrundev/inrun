# Profiles 09 — User-Defined via `spec.imports`

One CR. One Deployment. Three NetworkPolicies. Two HPAs. Every profile name is owned by the team — none are Orkestra built-ins. The profiles live in `motifs/motif.yaml` and are imported at the Katalog level via `spec.imports`, making them available to every CRD without re-declaring them inline.

**What you learn:** how to declare all six user-defined profile classes in a Motif; how `spec.imports` makes profiles available Katalog-wide; that `ork validate` enforces references against your imported profile registry; that profile names can be anything meaningful to your team.

---

## How it works

Profiles are declared once in `motifs/motif.yaml` and imported via `spec.imports`:

```yaml
spec:
  imports:
    - motif: ./motifs/motif.yaml

  crds:
    service:
      operatorBox:
        reconcile:
          onCreate:
            deployments:
              - resources:
                  profile: team-api        # from the imported motif
                securityContext:
                  profile: team-baseline   # from the imported motif
```

Only `profiles:` from the Motif is consumed at `spec.imports`. Resources, status, and admission in the Motif are ignored at this level.

---

## Profiles declared in `motifs/motif.yaml`

| Class | Name | Purpose |
|---|---|---|
| `networkPolicies` | `team-deny-all` | Block all ingress and egress |
| `networkPolicies` | `team-allow-internal` | Allow ingress from pods in the same namespace |
| `networkPolicies` | `team-allow-dns` | Allow outbound UDP/TCP 53 for DNS |
| `hpa` | `team-conservative` | 70% CPU, scale-down one pod per minute |
| `hpa` | `team-responsive` | 50% CPU, scale-down 25% per 15 seconds |
| `resources` | `team-api` | 200m/128Mi requests, 1/512Mi limits |
| `probes` | `team-standard` | 10s delay, 15s period, 3 failures |
| `containerSecurity` | `team-baseline` | Drop NET_RAW, no privilege escalation |
| `podSecurity` | `team-nonroot` | runAsUser 1000, fsGroup 1000 |

---

## Step 1 — Validate

Validate and inspect the full merged profile registry:

```bash
ork validate --profiles
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

Verify the resources:

```bash
kubectl get deployment,networkpolicies,hpa
```

Verify the resource profile was applied to the Deployment:

```bash
kubectl get deployment my-service -o jsonpath='{.spec.template.spec.containers[0].resources}' && echo
```

Verify the probe profile was applied:

```bash
kubectl get deployment my-service -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}' && echo
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
