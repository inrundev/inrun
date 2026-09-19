# 03 — TypeRegistry generation

The TypeRegistry generator is the most important generator — it wires Go types and hook/constructor functions into Orkestra's runtime. Without it, typed CRDs are unknown to the runtime and the reconciler will fail to start.

```sh
ork generate registry -f katalog.yaml
```

Output: `pkg/typeregistry/zz_generated_typeregistry.go`

## What the runtime needs

The Orkestra runtime uses four registries to look up types at startup:

| Registry | Maps | Used by |
|----------|------|---------|
| `ObjectRegistry` | GVK → `runtime.Object` factory | Decoding incoming CRs |
| `ListRegistry` | GVK → `client.ObjectList` factory | Listing CRs for reconcile |
| `HookRegistry` | GVK → hook closure | Calling user-supplied hook functions |
| `ReconcilerRegistry` | GVK → reconciler constructor | Calling user-supplied reconciler constructors |

The generated file populates whichever registries are needed for the CRDs declared in the Katalog.

## Three generation modes

### 1. Typed CRD (`apiTypes.location` is set)

The CRD declares Go types for its object and list:

```yaml
crds:
  myApp:
    apiTypes:
      object: MyApp
      list: MyAppList
      location: github.com/myorg/myoperator/pkg/types
```

Generated output registers both factories:

```go
orktypes.ObjectRegistry["myorg/v1/MyApp"] = func() runtime.Object { return &types.MyApp{} }
orktypes.ListRegistry["myorg/v1/MyApp"]   = func() client.ObjectList { return &types.MyAppList{} }
```

Also generates `RegisterScheme()` which calls each package's `AddToScheme` — required for the runtime to decode CRs by GVK.

### 2. Go hooks (`reconcile.hooks` is declared)

The CRD delegates part of the reconcile to a user-supplied Go function:

```yaml
crds:
  myApp:
    operatorBox:
      reconcile:
        hooks:
          location: github.com/myorg/myoperator/hooks
          function: Reconcile
```

Generated output registers the hook:

```go
orktypes.HookRegistry["myorg/v1/MyApp"] = func(ctx context.Context, ...) error {
    return hooks.Reconcile(ctx, ...)
}
```

### 3. Custom constructor (`reconcile.default: false`)

The CRD owns its entire reconcile loop:

```yaml
crds:
  myApp:
    operatorBox:
      reconcile:
        default: false
        constructor:
          location: github.com/myorg/myoperator/reconciler
          function: New
```

Generated output registers the constructor:

```go
orktypes.ReconcilerRegistry["myorg/v1/MyApp"] = func(...) (orkreconciler.Reconciler, error) {
    return reconciler.New(...)
}
```

## Import deduplication

When multiple CRDs share a location, the generator deduplicates imports and assigns stable aliases. If two packages would produce the same alias (e.g. two `types` packages from different module paths), `resolveAlias` appends a numeric suffix to guarantee uniqueness.

## Skipping generation

If all enabled CRDs in the Katalog are dynamic-template CRDs (no `apiTypes.location`, no hooks, no constructor), `TypeRegistry` skips file generation entirely — the `GenericReconciler` handles these at runtime without any registered types.

→ Next: [04-crd-generation.md](04-crd-generation.md)
