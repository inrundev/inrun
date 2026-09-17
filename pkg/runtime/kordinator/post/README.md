# post

Runs the postReconcile phase for every reconcile cycle.

Called by Kordinator after the reconciler returns (success or error). All operations are best-effort and never requeue.

## Responsibilities

- **Status** — writes the Ready condition (Layer 1) and declarative status fields (Layer 2); skips patch when nothing changed
- **Emit** — fires named Kubernetes events declared in `operatorBox.emit.events` when their conditions evaluate to true

## Entry point

```go
post.Apply(ctx, post.Input{...}, obj, resolver, box, reconcileErr, valResult)
```
