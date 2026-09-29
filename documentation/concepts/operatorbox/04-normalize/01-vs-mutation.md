# Normalize vs Mutation

Both phases run before reconciliation. They look similar in the Katalog but serve different purposes.

| | `normalize` | `mutation` |
|---|---|---|
| **Purpose** | Reshape a field — accept multiple valid formats, produce one canonical form | Supply a default — provide a value when the field is absent |
| **Triggered when** | Always, on every reconcile | When the declared field is missing or empty |
| **Sees** | The raw CR spec | The already-normalized spec |
| **Typical input** | Field is present but in the wrong shape | Field is absent entirely |
| **Writes to etcd** | Never — in-memory only | Via admission webhook (when enabled) |
| **Runs at** | Step 7 in the reconcile pipeline | Step 8 |

**Normalize handles shape. Mutation handles absence.**

A field that can arrive as either a string or a map is a normalize problem. A field that should default to `3` when the user omits it is a mutation problem. They compose — normalize first, mutation fills in anything still missing:

```yaml
normalize:
  spec:
    schedule: "{{ cronFromAny .spec.schedule }}"   # string or map → canonical string

admission:
  mutation:
    rules:
      - field: spec.concurrencyPolicy
        default: "Allow"
      - field: spec.successfulJobsHistoryLimit
        default: 3
```

---

## Defaulting inside normalize

The `default` note brings mutation's capability directly into normalize. This means you can handle both shape normalization and field defaults in a single phase — without deploying the Orkestra Gateway:

```yaml
normalize:
  spec:
    schedule:   "{{ cronFromAny .spec.schedule }}"
    environment: '{{ default "production" .spec.environment | toLower }}'
    replicas:    "{{ default 1 .spec.replicas }}"
    resources.requests.cpu:    '{{ resourceCPU . | default "100m" }}'
    resources.requests.memory: '{{ resourceMemory . | default "128Mi" }}'
```

- If `.spec.environment` is absent → `"production"`
- If `.spec.environment` is `"STAGING"` → `"staging"`
- If `.spec.replicas` is absent → `1`
- If `.spec.resources.requests.cpu` is absent → `"100m"`

This is the same outcome as webhook-based mutation defaults, enforced at reconcile time, with no Gateway required. The difference from `mutation:` rules:

| | `default` in normalize | `mutation` rules |
|---|---|---|
| Gateway required | No | Yes (for admission-time enforcement) |
| Can also reshape the value | Yes — compose with any note | No — defaults only |
| Visible at apply time | No — reconcile time only | Yes — admission response |
| Writes to etcd | Never | Yes (via admission webhook) |

Use `mutation:` rules when you need the default to be stored in etcd and visible to external tools reading the CR directly. Use `default` in normalize when in-memory consistency across reconcile is sufficient.

**Try it — apply a minimal CR with no optional fields, watch normalize fill them all in:**
```bash
ork init --pack use-cases
cd normalize/03-defaults-without-webhook
ork run
kubectl apply -f cr-minimal.yaml   # only spec.image declared
kubectl get workload api-server -o yaml | grep -A10 "status:"
# replicas, cpu, memory, concurrencyPolicy all defaulted
```

---

<!-- **Next →** [Patterns](02-patterns.md) -->
