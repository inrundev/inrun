# 02 — Kinds: Katalog vs Komposer

The merger supports two document kinds. They have distinct roles and constraints.

## Katalog

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: platform-crds
security:
  deletionProtection:
    enabled: true
notification:
  teams:
    platform:
      slack: { webhook: "$SLACK_WEBHOOK" }
spec:
  crds:
    myresource:
      enabled: true
      ...
```

**Rules:**
- Declares CRDs directly in `spec.crds`.
- Must NOT declare `imports:` — imports are a Komposer concern. The merger returns an error if an `imports:` block is present.
- All top-level fields (`security`, `notification`) apply to the CRDs declared in that file.

## Komposer

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform
imports:
  files:
    - url: https://raw.githubusercontent.com/org/repo/main/katalog.yaml
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 1.2.0
security:
  deletionProtection:
    enabled: true   # overrides imported Katalog setting if different
spec:
  crds:
    localoverride:  # inline CRDs — merged last, win on name conflict
      enabled: true
      ...
```

**Rules:**
- Composes Katalogs from multiple imports (file, Helm, registry).
- May declare inline `spec.crds` as local overrides — merged last, win on name conflict.
- Top-level fields (`security`, `notification`) from all imported Katalogs are accumulated. The Komposer's own block wins on conflict.
- A Komposer cannot reference another Komposer as an import — only `kind: Katalog` files are valid import targets.

## Dispatch

`loadKatalogFile` reads the `kind:` field after parsing, then dispatches:

```go
switch doc.Kind {
case "Katalog":  return m.loadKatalog(path, doc)
case "Komposer": return m.loadKomposer(path, doc)
}
```
