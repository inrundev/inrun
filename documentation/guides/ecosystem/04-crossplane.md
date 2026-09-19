# Crossplane

## Infra → Composite Claim

`03-crossplane` wraps Crossplane. Your team creates an `Infra` CR. Orkestra maps it to a Crossplane Composite Claim. Crossplane provisions the database. The developer wrote `type: postgres` and `size: medium` — they did not write a `compositionSelector`, a `storageGB` value, or a `connectionSecretRef` name.

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/03-crossplane
```

---

## The mapping

```text
Infra CR (internal)     Crossplane Composite Claim (ecosystem)
───────────────────     ──────────────────────────────────────
spec.type          →    kind: PostgreSQLInstance
spec.size          →    spec.parameters.storageGB (medium → 50)
spec.region        →    spec.parameters.region
spec.team          →    spec.compositionSelector.matchLabels.team
                   →    spec.writeConnectionSecretToRef.name: <name>-conn
```

The `storageGB` mapping (`small=20`, `medium=50`, `large=200`) is a platform decision encoded in the Katalog. The developer uses a human-readable size tier.

---

## The approval gate

This example adds a pattern not in `00-argocd` or `01-cert-manager`: a reconcile-time approval gate.

The `Infra` CR can be created without approval. It exists, it is visible in the Control Center, but no Crossplane Claim is created until `spec.approved: true` is patched:

```yaml
# Create the Infra CR — documents intent, creates nothing
kubectl apply -f cr.yaml

# Approve — Crossplane Claim is created, provisioning begins
kubectl patch infra webapp-db --type=merge -p '{"spec":{"approved":true}}'
```

This is enforced with a `when:` condition:

```yaml
operatorBox:
  reconcile:
    onCreate:
      customResources:
        - apiVersion: database.myorg.io/v1alpha1
          kind: PostgreSQLInstance
          when:
            - field: spec.approved
              equals: "true"
          ...
```

The Claim is only created when the condition is satisfied. On every reconcile cycle Orkestra re-evaluates the condition — if it is no longer met, the Claim is removed.

---

## Two enforcement points

Orkestra enforces at two points, and the second is a backstop for the first:

**Apply time** — the admission webhook evaluates validation rules and `when:` conditions synchronously during `kubectl apply`. A bad `spec.region`, a missing team label, or `spec.approved: false` blocks the request before the CR reaches etcd.

**Reconcile time** — every reconcile cycle re-runs the same validation rules and re-evaluates all `when:` conditions against the live CR. This catches anything that slipped past the webhook and reacts to field changes after admission: if `spec.approved` is patched back to `false`, the condition fails on the next cycle and the Claim is removed.

---

## Try it

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/03-crossplane
# Follow steps in README
```

→ [04 — Platform stack](./05-platform-stack.md) — all four tools in one Komposer.
