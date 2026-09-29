# Args — One Binary, Many Behaviours

`args:` lets a single reconcile implementation behave differently across environments, tiers, or gateway surfaces — with all variation declared in the Katalog.

---

## What args replace

Without args, variation means code changes: build flags, environment variables read at startup, separate binaries per tier, configuration files baked into images. Each approach ties deployment variation to code or build decisions.

With args, the hook or constructor implementation stays the same everywhere. The Katalog declares what values to pass. The same binary runs in staging and production, in an EU cluster and a US cluster, for a free-tier customer and an enterprise customer — and behaves differently because the args differ, not the code.

---

## Static and dynamic values

Args have two modes that compose freely:

**Static** values — strings, integers, booleans without `{{ }}` — are fixed at startup. Every CR reconciled by that operator sees the same value. Tier names, region identifiers, feature flags, resource limits: these belong here.

**Dynamic** values — strings containing `{{ }}` — are evaluated per-CR at reconcile time using the full expression language, including user-defined notes. The expression has access to the CR's spec, status, and metadata, and to the intent payload when the CR arrived via the gateway.

```yaml
args:
  tier: enterprise              # static — same for every CR
  maxReplicas: 50               # static
  region: "eu-west-1"           # static

  tenantId: "{{ .spec.tenantId }}"               # dynamic — per-CR field
  inBusinessHours: '{{ inBusinessHours }}'        # dynamic — note function
  featureEnabled: "{{ .external.flags.body }}"    # dynamic — from external call
```

Orkestra evaluates any string containing `{{ }}` at any depth in the args map. The hook or constructor receives the fully resolved values — it never sees template syntax.

---

## Hook args and constructor args

Args work the same way for hooks and constructors, with one difference in timing.

**Hook args** are evaluated fresh at each reconcile event. The hook receives values for the specific CR at that specific moment. A business-hours flag re-evaluates every cycle. A per-CR field that changes between reconciles reflects the new value. Hook args are for anything that needs to track CR state or time.

**Constructor args** are evaluated once when the operator starts, before any CR is reconciled. They are best for configuration that is stable for the lifetime of the operator — tier settings, region identifiers, resource limits. A constructor reads its args and builds a reconciler configured for the environment it is running in.

---

## Args and targets

Per-target declarations can carry their own `args:`, which are merged with the CRD-level args. Target-specific args override the keys they declare; all other keys are inherited. This lets the same hook or constructor serve multiple gateway surfaces from one Katalog — the surface determines the args, and the args determine the behaviour.

```yaml
operatorBox:
  reconcile:
    hooks:
      args:
        featureEnabled: "{{ .external.flags.body }}"   # CRD-level default
serve:
  target:
    v2-enabled:
      operatorBox:
        reconcile:
          hooks:
            args:
              featureEnabled: "true"   # always on for this surface
    v2-disabled:
      operatorBox:
        reconcile:
          hooks:
            args:
              featureEnabled: "false"  # always off for this surface
```

The hook reads `featureEnabled` and acts on it. It has no awareness of which target surface the CR arrived from. The Katalog determines what value it sees.

See [Reconcile Strategies](05-targets.md) for the full picture of per-target reconcile strategies.

---

## Related topics

- [Typed Operators — Reusability](../typed-operators/06-reusability.md) — one binary serving multiple CRD kinds and multiple environments via args
- [OperatorBox](../operatorbox/index.md) — where args are declared and how they are resolved
- [Reconcile Strategies](05-targets.md) — per-target args and full operatorBox variation
