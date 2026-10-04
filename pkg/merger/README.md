# pkg/merger

The merger package resolves and merges Katalog/Komposer YAML files into a single, unified set of CRD definitions. It is the ingestion layer that sits between the raw YAML on disk (or a remote source) and the runtime `Katalog` struct that drives the operator.

## What lives here

| File | Role |
|------|------|
| `merger.go` | `Merger` struct, `New`, `Merge`, `Enabled`, `All`, `Get`, `ToSpec`, `ToSecurity`, `ToUI` |
| `file.go` | `loadKatalogFile` dispatcher; `loadKatalog` (Katalog kind); `loadKomposer` (Komposer kind — imports + inline merge + accumulation) |
| `parse.go` | `parseKatalogDoc` — YAML decode + kind validation |
| `file_auth.go` | `loadSourceFileWithAuth` — HTTP/S fetch with bearer/basic auth; local file read |
| `helm.go` | `loadHelmSource` — Helm chart render → Katalog template extraction |
| `registry.go` / `registry_v2.go` | `loadRegistrySource` — fetch CRDs from the Orkestra registry |
| `helper.go` | `mergeKatalogSecurity`, `checkDuplicate`, `resolveEnvVar`, `writeTempFile`, `gitClone` |
| `helm_cache.go` | Disk cache for git and remote Helm sources (`~/.orkestra/helm/`) + `WarmHelmSource` |

## Merge rules

```
Katalog  (kind: Katalog)
  Declares CRDs directly in spec.crds.
  Must NOT declare imports — imports are a Komposer concern.

Komposer (kind: Komposer)
  Resolves imports (registry → files → helm) in that order.
  Inline spec.crds are merged last and win on name conflict.
  Top-level fields such as security are accumulated
  across all imports; the Komposer's own block wins on conflict.
```

### Deduplication scopes

| Scope | Mechanism |
|-------|-----------|
| Within one file's source tree | `localSeen map[string]string` |
| Across entry-point files | `seen map[string]string` in `Merge()` |
| Inline over source | valid — triggers `mergeCRDEntry` |
| Inline over inline | always an error |

### Top-level field accumulation

When a Komposer references multiple source Katalogs, the merger accumulates their top-level fields so that `ork generate rbac` and `ork generate configmap` against a Komposer produce the same output as running against the source Katalogs directly:

| Field | Accumulation strategy |
|-------|-----------------------|
| `security` | `mergeKatalogSecurity` — non-nil pointer fields in override win |

## Usage

```go
m := merger.New("katalog.yaml", "overrides.yaml")
if err := m.Merge(); err != nil { ... }

kat.Spec         = m.ToSpec()
kat.Security     = m.ToSecurity()
```

All `To*` methods panic if called before `Merge()`.

## Developer documentation

Full step-by-step documentation is in [docs/](docs/README.md).

| I want to… | Go to |
|-----------|-------|
| Understand the full merge pipeline | [01 — Architecture](docs/01-architecture.md) |
| Understand Katalog vs Komposer rules | [02 — Kinds](docs/02-kinds.md) |
| Add or understand a source type | [03 — Imports](docs/03-imports.md) |
| Debug duplicate CRD name errors | [04 — Deduplication](docs/04-deduplication.md) |
| Understand security inheritance | [05 — Top-Level Accumulation](docs/05-top-level-accumulation.md) |
| Understand Helm and file source caching | [07 — Source Caching](docs/07-source-caching.md) |
