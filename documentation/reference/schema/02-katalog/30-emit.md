# operatorBox.emit

`operatorBox.emit` declares outputs the runtime produces on behalf of the operator after each reconcile cycle. It groups two concerns: the `status:` declaration (moved here from the operatorBox top level) and `events:`, which declares named Kubernetes events emitted in `postReconcile`.

Both typed and declarative reconcilers have access to `emit:` — it is evaluated by the runtime, not the reconciler.

## Declaration

```yaml
spec:
  crds:
    app:
      operatorBox:
        emit:
          status:
            fields:
              - path: phase
                value: "{{ .status.phase }}"

          events:
            DatabaseReady:
              type: Normal
              reason: Ready
              message: "{{ .spec.name }} is ready"
              when:
                - field: .status.phase
                  equals: Ready

            DatabaseSyncFailed:
              type: Warning
              reason: SyncFailed
              message: "{{ .spec.name }} failed to sync: {{ .status.lastError }}"
              when:
                - field: .status.phase
                  equals: Failed
```

## `emit` fields

| Field     | Type          | Required | Description                                              |
| --------- | ------------- | -------- | -------------------------------------------------------- |
| `status`  | `StatusConfig`| no       | Declarative status fields written after every reconcile. |
| `events`  | map           | no       | Named Kubernetes events emitted in postReconcile.        |

## `emit.events[]` fields

`events` is a map. The map key is the event name — a stable identifier used for per-event deduplication within a reconcile cycle.

| Field     | Type          | Required | Description                                                                               |
| --------- | ------------- | -------- | ----------------------------------------------------------------------------------------- |
| `type`    | string        | yes      | Kubernetes event type. Must be `Normal` or `Warning`.                                    |
| `reason`  | string        | yes      | Event reason field. Static string.                                                        |
| `message` | string        | yes      | Human-readable message. Evaluated as a Go template against the prepared resolver context. |
| `when`    | `[]Condition` | no       | AND conditions. All must pass for the event to be emitted.                                |
| `or`      | `[]Condition` | no       | OR conditions. Any passing condition emits the event.                                     |

When neither `when:` nor `or:` is declared, the event is emitted on every reconcile cycle. Kubernetes event deduplication coalesces repeated identical events within the API server's dedup window, but `when:` should almost always be declared to avoid noise.

## Condition evaluation

`when:` and `or:` use the same condition syntax as `preReconcile.reconcileGate`. The resolver available in `postReconcile` includes cross, profiles, notes, and the status produced by the completed reconcile cycle — so conditions like `field: .status.phase` reflect current state.

See [When Conditions](06-when-conditions.md) for the full condition syntax reference.

## Pub/sub pairing with observe.events

`emit.events` is the producer side. `observe.events` is the consumer side. An event emitted by Operator A can trigger reconciliation in Operator B:

```yaml
# Operator A — emitting
operatorBox:
  emit:
    events:
      DatabaseSyncFailed:
        type: Warning
        reason: SyncFailed
        message: "{{ .spec.name }} sync failed"
        when:
          - field: .status.phase
            equals: Failed
```

```yaml
# Operator B — consuming
operatorBox:
  observe:
    events:
      dbFailed:
        reason: SyncFailed
        type: Warning
        reportingController: orkestra-runtime
```

Operator A emits the event. It lands in the Kubernetes event stream. Operator B's `observe.events.dbFailed` watch sees a `Warning` event with `reason: SyncFailed` from `orkestra-runtime` and enqueues Operator B's primary CR. Neither operator knows about the other.

All events emitted via `emit.events` are attributed to `reportingController: orkestra-runtime` — set by the runtime's event recorder.
