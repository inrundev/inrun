# Reconciler Profile

A reconciler profile is a named preset that expands into reconciler tuning — workers, resync interval, and queue depth — at Katalog load time.

You write a name. Orkestra writes the numbers.

---

## Built-in profiles

| Profile | workers | resync | queue.maxDepth | Use for |
|---------|---------|--------|----------------|---------|
| `high-throughput` | 10 | 5m | 1000 | High-volume operators managing hundreds of CRs under sustained load |
| `conservative` | 2 | 1m | 100 | Production operators where CRs change infrequently; reduces API server pressure |
| `development` | 1 | 30s | 50 | Local development; single worker makes log ordering predictable |

---

## Usage

Set `reconcile.profile` on any CRD entry:

```yaml
spec:
  crds:
    orders:
      operatorBox:
        reconcile:
          profile: conservative
```

The profile name is resolved at `ork validate` time. An unknown name is a hard error.

---

## Inline fields override the profile

Declare any inline field on `operatorBox.reconcile` to override the corresponding profile value. The rest of the profile still applies:

```yaml
operatorBox:
  reconcile:
    profile: conservative
    workers: 6    # override — profile's workers (2) ignored; resync and maxDepth come from conservative
```

Resolution order per field: inline → profile → global default.

---

## User-defined profiles

Built-ins cover the common cases. Declare your own in `profiles.reconciler` when your team has specific tuning requirements:

```yaml
profiles:
  reconcile:
    - name: api-service
      description: Balanced for a standard web service operator
      workers: 4
      resync: 30s
      queue:
        maxDepth: 200

    - name: batch-worker
      description: Low churn, background processing
      workers: 2
      resync: 5m
      queue:
        maxDepth: 500
spec:
  crds:
    orders:
      operatorBox:
        reconcile:
          profile: api-service
    archive:
      operatorBox:
        reconcile:
          profile: batch-worker
```

User-defined profiles are checked before built-ins. Declaring a profile named `conservative` shadows the built-in and logs a warning at validate time.

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| Many CRs, high create/update rate | `high-throughput` |
| Production operator, CRs change rarely | `conservative` |
| Local development or CI loop | `development` |
| Team-specific tuning | User-defined |

---

## How it differs from other profile classes

All other profile classes control **what child resources are created** — they expand into Deployment resource limits, NetworkPolicy rules, HPA scaling behavior. Reconciler profiles control **how the operator itself runs** — how many goroutines process CRs, how often it re-enqueues them, and how deep the queue can grow before backpressure kicks in.

This means reconciler profiles are set once per CRD entry at the `operatorBox.reconcile` level, not on individual child resource entries.

---

→ Back: [10 — User-Defined Profiles](10-user-defined-profiles.md) | [Profiles index](index.md)
