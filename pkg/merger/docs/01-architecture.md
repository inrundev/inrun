# 01 — Architecture

## Pipeline overview

```
CLI / konstructRuntime
    │
    │  one or more file paths (--file flags or comma-separated)
    ▼
merger.New(paths...)
    │
    │  .Merge()
    ▼
┌─────────────────────────────────────────┐
│  for each entry-point path              │
│                                         │
│  loadKatalogFile(path)                  │
│    │                                    │
│    ├── kind: Katalog → loadKatalog      │
│    │     reads spec.crds, sets          │
│    │     m.security                     │
│    │     as side-effects                │
│    │                                    │
│    └── kind: Komposer → loadKomposer    │
│          resolves imports in order:     │
│          1. registry imports            │
│          2. file imports                │
│          3. helm imports                │
│          4. inline spec.crds            │
│          accumulates top-level fields   │
│          from each source               │
└─────────────────────────────────────────┘
    │
    │  duplicate check across entry-point files
    ▼
m.result  (map[string]CRDEntry, all imports merged)
m.security (accumulated)

    │
    │  callers consume via:
    ▼
m.ToSpec()         → orktypes.KatalogSpec
m.ToSecurity()     → orktypes.KatalogSecurity
m.Enabled()        → map[string]CRDEntry  (enabled only)
```

## Key invariants

- `Merge()` is called exactly once. All `To*` and query methods panic if called before it.
- The merger is not thread-safe during `Merge()`. After it returns, reads from `Enabled`, `All`, `Get` are safe without synchronisation because the result map is never mutated.
- A Komposer may not reference another Komposer as an import — only Katalog kind is valid inside `imports:`. This prevents unbounded recursion.
- CRD names must be globally unique across all imports. A duplicate is always an error; it is never silently resolved.
