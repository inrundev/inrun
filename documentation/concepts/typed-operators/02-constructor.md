# Constructor

A constructor replaces the Generic Reconciler entirely. Your Go code owns the full reconcile loop — declarative templates are not applied when `default: false`.

Use a constructor when:

- **Migrating an existing controller-runtime operator** — change the `Reconcile` signature from `(ctx, req ctrl.Request) (ctrl.Result, error)` to `(ctx context.Context, req domain.Request) (domain.Result, error)`, remove the manager setup, and register the constructor in the Katalog. The informer, workqueue, worker pool, leader election, metrics, and panic recovery are all provided by Orkestra. Your reconcile logic is unchanged.
- **Running a custom state machine** — when the reconcile loop itself is stateful and not easily expressed as declarative templates with `when:` conditions.

For new operators, prefer [hooks in hybrid mode](./01-hooks.md#hybrid). Only reach for a constructor when you need to own the full loop.

---

## Katalog

```yaml
spec:
  crds:
    pipeline:
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Pipeline
        plural: pipelines
        object: Pipeline
        objectList: PipelineList
        location: github.com/myorg/pipeline-operator/api/v1alpha1

      operatorBox:
        reconcile:
          default: false   # disable Generic Reconciler; constructor owns everything
          workers: 5
          resync: 10s

          constructor:
            location: github.com/myorg/pipeline-operator/reconciler@v2.0.0
            function: NewPipelineReconciler
            resources:
              - kind: Job
                group: batch
                version: v1
                plural: jobs
            args:
              maxRetries: 3
              timeoutSeconds: 300
              notifyOnSuccess: true
```

`reconcile.default: false` tells Orkestra not to use the Generic Reconciler. The constructor at `location` provides the complete reconcile implementation.

The `@version` suffix in `location` is shorthand for the `version:` field — `location: github.com/myorg/reconciler@v2.0.0` is equivalent to declaring `version: v2.0.0` separately. Both forms are accepted; the `@` shorthand keeps the declaration compact.

`fetch` controls whether `ork generate registry` adds the module to your project:

| `fetch` | Behaviour |
|---|---|
| `false` (default) | Module must already be in `go.mod`. `ork generate registry` wires it without modifying dependencies. |
| `true` | `ork generate registry` runs `go get <location>@<version>` automatically, adding or updating the module in `go.mod` and `go.sum`. |

Use `fetch: true` when pulling the constructor from a remote module you have not yet added to the project. Use `fetch: false` (or omit it) when the module is already a local dependency.

`resources` declares what Kubernetes resources the constructor manages. It serves two purposes:

- **RBAC generation** — Orkestra generates `get/list/watch/create/update/patch/delete` permissions for each declared type.
- **Implicit watch informer** — Orkestra automatically starts a watch informer for each declared resource, the same as declaring an explicit `watch:` entry with all events and owner-reference key resolution. This means:
  - `r.client.Get` and `r.client.List` for that type are served from cache (no live API call after the informer syncs)
  - When an owned resource changes and has an ownerReference pointing to the primary CR, Orkestra re-enqueues that CR automatically

No extra YAML is needed. If you need finer control — custom event filters, field indexes, or a different key resolution strategy — declare an explicit `watch:` entry for that type. It takes priority over the implicit informer from `resources:`.

`args` passes configuration from the Katalog into the constructor. Orkestra attaches the args to the `kube` client before calling the constructor function — no extra wiring needed.

**String values support Go template expressions**, including strings inside nested maps — the resolver recurses into them. Integers and booleans have no template syntax (YAML parses them as native types) so they are always read as-is. Dynamic string values — those with `{{ }}` — need to be resolved per-CR at reconcile time. The constructor calls `kube.ScopedFor(resolver.TemplateEvaluator())` itself after building its resolver:

```go
func NewPipelineReconciler(kube kubeclient.Interface) domain.Reconciler {
    // Static args are safe to read at construction time — no templates involved.
    maxRetries      := kube.Args().Int("maxRetries")
    notifyOnSuccess := kube.Args().Bool("notifyOnSuccess")
    return &PipelineReconciler{
        kube:            kube,   // holds rawArgs; ScopedFor resolves them at reconcile time
        maxRetries:      maxRetries,
        notifyOnSuccess: notifyOnSuccess,
    }
}

func (r *PipelineReconciler) Reconcile(ctx context.Context, obj domain.Object) error {
    resolver := template.NewResolver(ctx, obj)
    kube := r.kube.ScopedFor(resolver.TemplateEvaluator())  // resolve {{ }} in args
    ns     := kube.Args().String("namespace")   // "prod" (from {{ .metadata.namespace }})
    source := kube.Args().String("source")      // "GITHUB" (from {{ upper .spec.source }})
    _ = ns; _ = source
    return nil
}
```

The full note FuncMap is available — `default`, `upper`, `lower`, and all other note functions work in constructor args exactly as they do in hook args and `onCreate` templates.

For structured access, bind the whole map to a typed struct with `kube.Args().BindArgs(&cfg)`. See the [schema reference](../../reference/schema/02-katalog/04-operatorbox.md) for the full `args` API.

!!! note Constructor Ownership
    `onCreate`, `onReconcile`, `onDelete`, `hooks`, and `status.fields` are all ignored when `default: false`. The constructor owns status management directly.

---

## Constructor function

The `function` value must be an exported function with this signature:

```go
package reconciler

import (
    "github.com/orkspace/orkestra/pkg/kubeclient"
    "github.com/orkspace/orkestra/domain"
)

func NewPipelineReconciler(kube kubeclient.Interface) domain.Reconciler {
    return &PipelineReconciler{kube: kube}
}
```

Orkestra calls this function once at startup and uses the returned `domain.Reconciler` for all reconcile events on this CRD.

The informer and event recorder are available via `kube.GetInformer()` and `kube.GetEventRecorder()` — injected by the runtime before the constructor is called. Constructor args are read via `kube.Args()`.

---

## Two styles of reconcile implementation

### Lift and change signature

The minimal migration from controller-runtime. Change the `Reconcile` signature and remove the manager setup. Your resource management logic stays exactly as it was.

**Before (controller-runtime)**:
```go
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    existing := &appsv1.Deployment{}
    err := r.Get(ctx, req.NamespacedName, existing)
    if errors.IsNotFound(err) {
        return ctrl.Result{}, r.Create(ctx, desired)
    }
    patch := client.MergeFrom(existing.DeepCopy())
    existing.Spec = desired.Spec
    return ctrl.Result{}, r.Patch(ctx, existing, patch)
}
```

**After (Orkestra constructor)**:
```go
func (r *WebAppReconciler) Reconcile(ctx context.Context, req domain.Request) (domain.Result, error) {
    key := req.Key // namespace/name — same content as the old req.String()
    existing := &appsv1.Deployment{}
    err := r.kube.Get(ctx, namespace, name, existing)
    if errors.IsNotFound(err) {
        return domain.Result{}, r.kube.Create(ctx, desired)
    }
    patch := client.MergeFrom(existing.DeepCopy())
    existing.Spec = desired.Spec
    return domain.Result{}, r.kube.Patch(ctx, existing, patch)
}
```

What you removed: `ctrl.NewManager`, `SetupWithManager`, scheme registration in `main.go`. Orkestra provides the informer, workqueue, worker pool, leader election, metrics, and panic recovery. You kept all the business logic.

---

### With Orkestra resources

Replace the manual Get / IsNotFound / Create / Patch pattern with the `pkg/resources` library. Each resource type exports four functions: `Create`, `Update`, `Delete`, `DeleteIfOwned`.

**Before (manual)**:
```go
existing := &appsv1.Deployment{}
err := r.kube.Get(ctx, namespace, name, existing)
if errors.IsNotFound(err) {
    return r.kube.Create(ctx, desired)
}
patch := client.MergeFrom(existing.DeepCopy())
existing.Spec = desired.Spec
return r.kube.Patch(ctx, existing, patch)
```

**After (Orkestra resources)**:
```go
import orkdeploy "github.com/orkspace/orkestra/pkg/resources/deployments"

spec := orkdeploy.Resolve(orktypes.DeploymentTemplateSource{
    Name:      obj.GetName(),
    Namespace: obj.GetNamespace(),
    Image:     typedObj.Spec.Image,
    Replicas:  fmt.Sprintf("%d", typedObj.Spec.Replicas),
}, obj.GetName())

return orkdeploy.Update(ctx, kube, obj, spec)
```

`Update` handles: create if absent, patch if drifted, owner references, system labels (`managed-by: orkestra`, `orkestra-owner: <cr-name>`). No conditional logic in your reconciler.

To remove a resource only if this CR owns it:

```go
orkdeploy.DeleteIfOwned(ctx, kube, obj, name, namespace)
```

`DeleteIfOwned` is a no-op if the resource does not exist or is owned by a different CR.

---

## Generate and build

After writing the Katalog, one command generates both `pkg/typeregistry/zz_generated_typeregistry.go` and `cmd/orkestra/main.go`:

```bash
ork generate registry --file katalog.yaml
go build ./cmd/orkestra
```

You write neither generated file. Re-run `ork generate registry` whenever you change `apiTypes`, `hooks`, or `constructor` declarations in the Katalog.

---

To try the full working example:

```bash
ork init --pack advanced
cd 10-constructor
```

Follow the steps in the README