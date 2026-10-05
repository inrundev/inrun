# constructor

A `Pipeline` operator whose reconciler is a full `domain.Reconciler`: it runs
Jobs in order (build → test → notify) and owns the whole loop. The same
behaviour can be declared in a Katalog; this shows the constructor pattern for
logic you want to keep in Go.

`reconciler/pipeline_reconciler.go` holds `NewPipelineReconciler` and `Reconcile`;
types are in `api/v1alpha1/`.

### 1. Generate the registry

Writes the type registry and `cmd/orkestra/main.go`.

```bash
make registry
```

### 2. Build your ork

Installs an `ork` with the Pipeline type compiled in at `~/.orkestra/bin/ork`.

```bash
make build
```

### 3. Validate

```bash
ork validate
```

### 4. Run

Applies `manifests/crd.yaml` and `manifests/cr.yaml`, then reconciles.

```bash
ork run
```

### 5. Check the result

In a second terminal.

```bash
kubectl get pipeline,jobs -w
```

### 6. Deploy

Build and push the image:

```bash
make release IMAGE=<registry>/<image>:<tag>
```

Apply the RBAC and Katalog bundle:

```bash
ork generate bundle -f katalog.yaml -o bundle.yaml
```

```bash
kubectl apply -f bundle.yaml
```

Install Orkestra with your image:

```bash
helm upgrade --install orkestra orkestra/orkestra \
  --set runtime.image.repository=<registry>/<image> \
  --set runtime.image.tag=<tag> \
  --namespace orkestra-system --wait
```

### 7. Clean up

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

### 8. E2E

Runs the example in a kind cluster with your image from the deploy step, since only it has the Go types compiled in.

```bash
ork e2e --set runtime.image.repository=<registry>/<image> --set runtime.image.tag=<tag>
```
