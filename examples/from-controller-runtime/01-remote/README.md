# 01 — Remote

The same WebApp operator. The reconciler is a bash script.

No Go. No image build. No Kubernetes SDK. No kubeconfig. Orkestra owns the queue, backoff, informer, owner references, SSA apply, and health. The script owns the Deployment and Service specs.

---

## What changed

| | controller-runtime | Remote Reconciler |
|---|---|---|
| Language | Go | anything that speaks HTTP |
| Kubernetes client | `controller-runtime/client` | none |
| Owner references | `metav1.NewControllerRef(...)` | automatic |
| RBAC | hand-written markers | automatic |
| Drift correction | not implemented | automatic (SSA, `forceConflict: true`) |
| Garbage collection | `Owns()` in `SetupWithManager` | owner refs set by Orkestra |
| Reconciler auth | n/a | Bearer token via `secretRef` |
| Build & deploy | Dockerfile, image push | `./reconciler.sh` |

---

## Step 1 — Install dependencies

The bash reconciler uses `socat` (HTTP loop) and `jq` (JSON). Skip if already available.

```bash
command -v socat jq >/dev/null 2>&1 || sudo apt-get install -y socat jq
```

---

## Step 3 — Start the reconciler

[`secret.yaml`](secret.yaml) ships with a default token. `ork run` applies it automatically — no manual setup needed. To use a different token, edit `secret.yaml` and pass the same value to the reconciler.

```bash
RECONCILER_TOKEN=orkestra-demo-token ./reconciler/reconciler.sh
```

The script listens on `:8025`. It reads the WebApp spec, builds a Deployment and Service as plain JSON, and returns them. No imports. No SDK. No cluster access.

---

## Step 4 — Run

```bash
ork validate
ork run
```

Orkestra installs the CRD, starts the informer, and on every `WebApp` event POSTs to the reconciler. The returned Deployment and Service are SSA-applied with owner references pointing at the WebApp CR.

---

## Step 5 — Verify

```bash
kubectl get webapp my-webapp -o jsonpath='{.status}' | jq .
kubectl get deployment my-webapp
kubectl get service my-webapp-svc
```

### Drift correction
Edit the `WebApp` spec — change `replicas` or `image` in the [`cr.yaml`](cr.yaml) file. The reconciler is called again, the Deployment is updated.

Apply and verify
```bash
kubectl apply -f cr.yaml
kubectl get deployment my-webapp
```


Manually edit the Deployment's replica count. On the next reconcile cycle Orkestra restores it. Drift correction is automatic with SSA.

---

## Step 6 — Delete

```bash
kubectl delete webapp my-webapp
```

The Deployment and Service are deleted automatically. Kubernetes GC cascades via the owner references Orkestra set.

---

## The contract

The reconciler receives:

```json
{
  "key":    "default/my-webapp",
  "gvk":    { "Group": "migration.demo.orkestra.io", "Version": "v1alpha1", "Kind": "WebApp" },
  "object": { ...WebApp CR (managedFields stripped)... },
  "args": {
    "logLevel":    "info",
    "environment": "development",
    "appName":     "my-webapp"
  }
}
```

`args` are declared in the katalog and evaluated against the CR at runtime — no parsing of the raw object needed for common values.

It returns:

```json
{
  "result":  "ok",
  "status":  { "phase": "Running", "endpoint": "...", "replicas": 1 },
  "resources": [
    { "type": "deployment", "fields": { "name": "my-webapp", "image": "nginx:latest", "replicas": 1, "port": 80 } },
    { "type": "service",    "fields": { "name": "my-webapp-svc", "port": 80, "targetPort": 80 } }
  ]
}
```

Resources use the intent form: `type` names the built-in kind and `fields` is a flat map of values. Orkestra constructs the full Kubernetes object, sets owner references, and SSA-applies it. The reconciler never writes Kubernetes schema.

That is the entire interface. Any language, any runtime, any host that can receive HTTP and return JSON is a valid operator.

---

## Endpoint adapts automatically

The katalog declares a `reconcilerEndpoint` note that picks the right URL based on where Orkestra is running:

```yaml
notes:
  functions:
    - name: reconcilerEndpoint
      expression: '{{ if .ork.inCluster }}http://webapp-reconciler.default.svc.cluster.local:8025/reconcile{{ else }}http://localhost:8025/reconcile{{ end }}'
```

`.ork.inCluster` is a runtime fact Orkestra injects into every template expression. `false` locally, `true` inside a cluster. The endpoint field then simply reads:

```yaml
endpoint: "{{ reconcilerEndpoint }}"
```

One katalog file. No separate variant for E2E or in-cluster deploys.

---

## Try it in Python

Stop the bash reconciler, then run the Python one instead. The katalog and the CR do not change.

```bash
RECONCILER_TOKEN=orkestra-demo-token python3 reconciler/main.py
```

Same Deployment. Same Service. Same status. Different language — zero other changes.

## Or Go

```bash
RECONCILER_TOKEN=orkestra-demo-token go run reconciler/main.go
```

Pure `net/http`. No controller-runtime. No kubeconfig. Still Go.

---

## Runtime facts

The `reconcilerEndpoint` note uses `.ork.inCluster` — one of three facts Orkestra injects into every template expression at startup.

| Key | Description |
|-----|-------------|
| `.ork.inCluster` | `true` when running inside a Kubernetes pod |
| `.ork.namespace` | Namespace Orkestra is deployed in |
| `.ork.version` | Running Orkestra version string |

Full reference: https://orkestra.sh/docs/reference/runtime-context/
