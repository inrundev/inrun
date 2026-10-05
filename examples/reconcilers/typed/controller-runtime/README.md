# controller-runtime

A controller-runtime `WebApp` reconciler running inside Orkestra with its
`Reconcile` method unchanged. Three things differ from a controller-runtime
project:

1. No `SetupWithManager`: Orkestra provides the informer, queue, workers,
   leader election and metrics.
2. No scheme setup: Orkestra registers the types at startup.
3. A constructor wires the reconciler in:

```go
func NewWebAppReconciler(kube kubeclient.Interface) domain.Reconciler {
    return domain.ReconcilerFrom(&WebAppReconciler{
        Client: orkadapter.ToClient(kube),
    })
}
```

`orkadapter.ToClient` gives the struct the `client.Client` it already expects.

### 1. Generate the registry

Writes the type registry and `cmd/orkestra/main.go`.

```bash
make registry
```

### 2. Build your ork

Installs an `ork` with the WebApp type compiled in at `~/.orkestra/bin/ork`.

```bash
make build
```

### 3. Validate

```bash
ork validate
```

### 4. Simulate

```bash
ork simulate
```

### 5. Run

Applies `manifests/crd.yaml` and `manifests/cr.yaml`, then reconciles.

```bash
ork run
```

### 6. Check the result

In a second terminal.

```bash
kubectl get webapps,deployments,services
```

### 7. Deploy

Build and push the image:

```bash
IMAGE_REPO=<registry>/<image> IMAGE_TAG=<tag> make release
```

Set the same image under `runtime.image` in `test/values.yaml`, then install
Orkestra with it:

```bash
helm upgrade --install orkestra orkestra/orkestra -f test/values.yaml \
  --namespace orkestra-system --wait
```

### 8. Clean up

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

### 9. E2E

Runs the example in a kind cluster with your image from the deploy step, since only it has the Go types compiled in.

```bash
ork e2e --set runtime.image.repository=<registry>/<image> --set runtime.image.tag=<tag>
```
