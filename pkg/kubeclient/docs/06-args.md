# 06 — Args

## What it is

`Args` is the typed view of the `args:` map declared under `reconciler.hooks.args` or
`reconciler.constructor.args` in katalog.yaml. It is attached to a `KubeClient` copy before
a hook or constructor is called, and read via `kube.Args()`.

## Two-phase model

Args go through two phases:

1. **Declaration** — values are stored as `rawArgs` when `WithArgs` is called at startup.
   String values may contain Go template expressions (`{{ }}`). Integers and booleans
   have no template syntax so they are stored as-is. Nested maps are stored as-is too,
   but ScopedFor recurses into them to evaluate any string values they contain.

2. **Resolution** — `ScopedFor(eval)` walks `rawArgs`, evaluates every string that
   contains `{{` using the provided evaluator, and returns a new `KubeClient` copy with
   the resolved values in `args`. Subsequent calls to `kube.Args()` return the resolved map.

```
katalog.yaml args:               WithArgs()           ScopedFor(resolver.TemplateEvaluator())
─────────────────────────────    ──────────────────── ────────────────────────────────────────
region: "{{ .spec.region }}"  →  rawArgs (unevaluated) →  args["region"] = "eu-west-1"
readReplicaCount: 2           →  rawArgs (pass-through) →  args["readReplicaCount"] = 2
```

## KubeClient interface

```go
type KubeClient interface {
    // ...
    Args() Args
    WithArgs(args Args) KubeClient
    ScopedFor(eval func(string) (string, bool)) KubeClient
}
```

## Args accessors

```go
type Args map[string]interface{}

func (a Args) String(key string) string          // "" if absent or wrong type
func (a Args) Bool(key string) bool              // false if absent or wrong type
func (a Args) Int(key string) int                // 0 if absent; handles int/int64/float64
func (a Args) Sub(key string) Args               // empty Args if absent
func (a Args) Slice(key string) []interface{}    // nil if absent
func (a Args) BindArgs(dst interface{}) error    // JSON round-trip into a struct
```

`Args()` always returns a non-nil map — absent keys return zero values, no nil checks needed.

## Resolution in practice

### Hooks (automatic)

`Generic Reconciler` calls `ScopedFor` after building the per-CR resolver, before the hook
runs. Hook authors read resolved values directly — no extra wiring:

```go
func onReconcile(ctx context.Context, obj *apiv1.Database) error {
    kube, _ := kubeclient.FromContext(ctx)
    region  := kube.Args().String("region")           // resolved from {{ .spec.region }}
    replicas := kube.Args().Int("readReplicaCount")   // static, passed through
    return nil
}
```

### Constructors (opt-in)

Constructor authors own their reconcile loop. They call `ScopedFor` themselves after building
their resolver, giving them the same capability with full control over timing:

```go
func (r *PipelineReconciler) Reconcile(ctx context.Context, obj domain.Object) error {
    resolver := template.NewResolver(ctx, obj)
    kube := r.kube.ScopedFor(resolver.TemplateEvaluator())
    ns     := kube.Args().String("namespace")    // resolved from {{ .metadata.namespace }}
    source := kube.Args().String("source")       // resolved from {{ upper .spec.source }}
    return nil
}
```

## Template support

String args have access to the full note FuncMap — the same functions available in
`onCreate`/`onReconcile` templates:

```yaml
args:
  region: '{{ default "us-east-1" .spec.region }}'
  source:  "{{ upper .spec.source }}"
  label:   "{{ .metadata.name }}-worker"
```

Integers and booleans have no template syntax — YAML parsed them as native types before
Go saw the value, so `ScopedFor` returns them as-is. Nested maps are recursed into:
every string inside a nested map is evaluated; the map container itself is not a template.

## ResolveArgsMap

The package-level `ResolveArgsMap` function is exported for implementations of `KubeClient`
outside the package (e.g. `pkg/simulate.FakeKubeclient`):

```go
func ResolveArgsMap(rawArgs map[string]interface{}, eval func(string) (string, bool)) Args
```
