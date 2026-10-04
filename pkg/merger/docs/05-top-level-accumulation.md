# 05 — Top-Level Field Accumulation

When a Komposer references multiple source Katalogs, each source may declare its own top-level `security:` block. It is accumulated so that `ork generate rbac` and `ork generate configmap` against a Komposer produce the same output as running them against the source Katalogs directly.

## The problem without accumulation

`loadKatalog` sets `m.security` as a side-effect when it loads each source Katalog. Without accumulation, the last source loaded overwrites all earlier ones. The Komposer's own (possibly empty) block then further overwrites whatever the last source set.

Result: the Komposer appears to have no security, even though its sources declare it.

## The solution

`loadKomposer` declares an accumulator before the source loops:

```go
var accSecurity orktypes.KatalogSecurity
```

After each source is loaded, the side-effects on `m` are captured:

```go
accSecurity = mergeKatalogSecurity(accSecurity, m.security)
```

At the end, the Komposer's own block is layered on top:

```go
m.security = mergeKatalogSecurity(accSecurity, doc.Security)
```

## Merge semantics

### `security` — `mergeKatalogSecurity(base, override)`

- Pointer fields (`DeletionProtection`, `Webhooks`, `Conversion`, `NamespaceProtection`): non-nil override wins; nil falls through to base.
- `ServiceName` string: non-empty override wins.
- Rationale: a Komposer should be able to tighten or loosen security without re-declaring every source field.
