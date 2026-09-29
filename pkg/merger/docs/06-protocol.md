# 06 — Protocol: what the merger accepts

The merger has a strict contract for what a valid Katalog or Komposer document looks like.
Every document that passes through the merger (local file, remote URL, Helm chart output, OCI pull)
is validated against these rules before any merging happens.

## Required fields

```yaml
apiVersion: orkestra.orkspace.io/v1   # enforced — hard error if missing or wrong
kind: Katalog                          # or Komposer — anything else is silently skipped
metadata:
  name: my-operator                    # required — empty name is a hard error
```

## apiVersion

Only `orkestra.orkspace.io/v1` is supported in v1. The merger returns a descriptive error
if the apiVersion is missing or unrecognised:

```
"katalog.yaml": unsupported apiVersion "orkestra.orkspace.io/v2"
  Supported: [orkestra.orkspace.io/v1]
  This usually means the pattern was built for a different version of Orkestra.
  Check the upstream pattern's katalog.yaml or update Orkestra.
```

When pulling from an external registry or Git source, a wrong apiVersion usually means
the upstream pattern targets a different Orkestra release. Fix: update Orkestra, or pin
the pattern version.

## spec.crds format

`spec.crds` is always a **map**. The CRD name is the map key:

```yaml
spec:
  crds:
    database:          # ← CRD name is the key
      enabled: true
      apiTypes:
        group: apps.example.io
        version: v1alpha1
        kind: Database
```

There is no list form. If a pulled pattern uses `- name: database` under `spec.crds`,
the merger detects it before YAML parsing and returns a clear error:

```
"katalog.yaml": spec.crds must be a map (name: {}) not a list (- name:).
```

One mental model: CRD names are unique keys, not ordered items. The map form
enforces this at the schema level.

## imports vs spec.crds

Imports are resolved before inline `spec.crds`. Inline CRDs win on name conflict:

```
registry imports  →  file imports  →  helm imports  →  spec.crds (wins)
```

Only `kind: Komposer` documents may declare `imports:`. A `kind: Katalog` with an
`imports:` block is a hard error.

## What the merger does NOT validate

- Whether CRD apiTypes point to real Kubernetes groups — that is a runtime concern.
- Webhook configuration correctness — validated by Kubernetes itself when applied.

## Adding a new apiVersion

When breaking changes require a new apiVersion:

1. Add the new version to `konfig.apiVersions` in `pkg/konfig/constants.go`.
2. Add migration logic to `parseKatalogDoc` in `parse.go` if old documents need rewriting.
3. Bump the version in `pkg/registry/constant.go`'s OCI media type if the OCI format changes.
4. Update this document.
