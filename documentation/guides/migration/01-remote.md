# Remote — Zero Build

The same WebApp operator. The reconciler is a bash script.

No Go. No image build. No Kubernetes SDK. No kubeconfig. Orkestra owns the queue, backoff, informer, owner references, SSA apply, and health. The script owns the Deployment and Service specs.

```bash
cd from-controller-runtime/01-remote
RECONCILER_TOKEN=orkestra-demo-token ./reconciler/reconciler.sh &
ork run
```

---

## What changed

| | controller-runtime | Remote |
|---|---|---|
| Language | Go | anything that speaks HTTP |
| Kubernetes client | `controller-runtime/client` | none |
| Owner references | `metav1.NewControllerRef(...)` | automatic |
| RBAC | hand-written markers | automatic |
| Drift correction | not implemented | automatic (SSA) |
| Garbage collection | `Owns()` in `SetupWithManager` | owner refs set by Orkestra |
| Build & deploy | Dockerfile, image push | `./reconciler.sh` |

---

## The contract

Orkestra POSTs to your endpoint on every watch event:

```json
{
  "key":    "default/my-webapp",
  "gvk":    { "Group": "...", "Version": "v1alpha1", "Kind": "WebApp" },
  "object": { ...WebApp CR... },
  "args":   { "logLevel": "info", "environment": "development" }
}
```

`args` are declared in the Katalog and evaluated against the CR — no raw object parsing needed for common values.

Your endpoint returns:

```json
{
  "result":  "ok",
  "status":  { "phase": "Running", "endpoint": "...", "replicas": 1 },
  "resources": [
    { "type": "deployment", "fields": { "name": "my-webapp", "image": "nginx:latest", "replicas": 1 } },
    { "type": "service",    "fields": { "name": "my-webapp-svc", "port": 80, "targetPort": 80 } }
  ]
}
```

Resources use the intent form — `type` names the built-in kind, `fields` is a flat map. Orkestra constructs the full Kubernetes object, sets owner references, and SSA-applies it. The reconciler never writes Kubernetes schema.

That is the entire interface. Any language, any runtime, any host that can receive HTTP and return JSON is a valid operator.

---

## Endpoint selection

The Katalog uses a note to pick the right URL depending on where Orkestra is running:

```yaml
notes:
  functions:
    - name: reconcilerEndpoint
      expression: '{{ if .ork.inPod }}http://webapp-reconciler.default.svc.cluster.local:8025/reconcile{{ else }}http://localhost:8025/reconcile{{ end }}'
```

`.ork.inPod` is `false` locally and `true` inside a pod. One Katalog file — no separate variant for local vs. in-cluster.

---

## Same operator, any language

The bash reconciler, a Python version, and a Go version all ship in the example. Swap them without touching the Katalog or the CR:

```bash
# Python
RECONCILER_TOKEN=orkestra-demo-token python3 reconciler/main.py

# Go (no controller-runtime, no kubeconfig)
RECONCILER_TOKEN=orkestra-demo-token go run reconciler/main.go
```

---

## When to use this

Pick remote when:
- The reconcile logic already exists as a service, a script, or a function
- You want to use a language other than Go
- You want to iterate on reconcile logic without a rebuild or redeploy cycle
- The operator's behaviour is simple enough that the HTTP overhead is irrelevant

---

## Next

- [Declarative](./02-declarative.md) — zero Go, zero HTTP, pure Katalog
- [Hybrid](./03-hybrid.md) — declare most resources, write Go for the rest
