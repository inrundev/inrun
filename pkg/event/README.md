# pkg/event

`event` wraps the Kubernetes event recorder so that reconcilers and control-plane components can emit `corev1.Event` objects without depending directly on `client-go/tools/record`.

```go
e := event.NewEvent(kube)
e.Start(ctx)

e.Eventf(obj, corev1.EventTypeNormal, "Reconciled", "created %s", name)
```

## Event recording

`Eventf` records events asynchronously — it spawns a goroutine per call and tracks in-flight work with a `sync.WaitGroup`. This prevents event recording from blocking the reconcile loop on API server latency.

Events are discarded silently once `Shutdown()` is called, so no goroutine leaks after the broadcaster is shut down.

## Shutdown

`Shutdown()` sets a stopped flag, waits for all in-flight events to flush (with the given context deadline as a timeout), then calls `broadcaster.Shutdown()` to release the underlying connection. A context timeout produces a warning log but is otherwise clean.

## Recorder interface

`Recorder` is declared in this package and is the only surface that reconcilers and constructors depend on:

```go
type Recorder interface {
    Eventf(obj runtime.Object, eventType, reason, messageFmt string, args ...interface{})
}
```

`*Event` implements `Recorder` for real clusters. Passing `nil` is safe anywhere that nil-coalesces internally (e.g. `NewGeneric Reconciler`). Constructor functions (`NewReconcilerFunc`) receive `event.Recorder` and should guard with `if ev != nil` before calling `Eventf`.
