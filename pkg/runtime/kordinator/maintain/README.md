# maintain

Applies labels, annotations, and finalizers to a CR on every reconcile cycle.

Called by Kordinator after `prepare` and before the reconciler is invoked.

## Responsibilities

- **Finalizers** — ensure box-declared finalizers are present; strip them in force-cleanup mode (`RemoveFinalizers=true`)
- **Labels** — managed marker, deletion-protection, strict-mode-exempt, user-defined template labels
- **Annotations** — managed-by and managed-since (write-once)

## Entry point

```go
err := maintain.Apply(ctx, maintain.Input{...}, obj, box, resolver)
```
