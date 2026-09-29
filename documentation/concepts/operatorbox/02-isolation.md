# Isolation and IPC

## Isolation guarantees

OperatorBoxes within the same binary do not share:

- Informer caches
- Workqueues
- Reconciler instances
- Health state
- Panic domains (each reconcile is wrapped in `safeReconcile`)

A panic in one operatorBox is recorded as a reconcile failure and triggers a requeue. It does not affect any other operatorBox. Queue pressure in one operatorBox does not affect processing latency in another.

The isolation is within a single OS process. OperatorBoxes share the Go runtime scheduler and memory allocator. They do not share any Orkestra-level data structures.

---

## Cross-operatorBox communication (IPC)

An operatorBox can observe another operatorBox's CR state through the `cross:` declaration. This is explicit, read-only, and zero-cost for same-binary operatorBoxes.

```yaml
operatorBox:
  reconcile:
    onReconcile:
      deployments:
        - name: "{{ .metadata.name }}"
          when:
            - field: "{{ phase .cross.db }}"
              equals: "Ready"
  observe:
    cross:
      - crd: managed-database
        selector:
          name: "{{ .metadata.name }}-db"
        as: db
```

The `cross:` declaration resolves through the `KatalogRegistry`, which holds a reference to every operatorBox's informer. Reading another operatorBox's state is an in-memory map lookup — the API server is not involved.

For cross-binary or cross-cluster observation, add a `source:` block with `host` and `type` — Orkestra constructs the URL automatically. For non-Orkestra APIs that expose the same JSON shape, use `source.endpoint` directly instead. See [ONCOP](../oncop/) for the full protocol.

---

## Why explicit IPC matters

Implicit sharing is how distributed systems become hard to reason about. When you can't tell which operators are reading each other's state, every change risks an invisible dependency.

Orkestra's `cross:` declaration makes the dependency visible: it appears in the Katalog, it's validated at load time, and it's visible in the Control Center. If the observed operatorBox is removed from the Katalog, the reference fails at load — not at runtime, not in production.
