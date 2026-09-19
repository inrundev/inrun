# Importing a Motif

Motifs are imported at the CRD entry level inside a Katalog, via the `imports:` field. Each import declares the source and binds CR field values to the Motif's inputs via `with:`.

---

## Basic import

```yaml
spec:
  crds:
    database:
      apiTypes:
        group: apps.example.io
        version: v1
        kind: Database
      imports:
        - motif: oci://ghcr.io/orkspace/patterns/motifs/postgres:v0.1.0
          with:
            image: "{{ .spec.image }}"
            volumeSize: "{{ .spec.storage | default \"10Gi\" }}"
            replicas: "{{ .spec.replicas | default \"1\" }}"
```

---

## Import sources

| Form | Example |
|------|---------|
| Local file | `motif: ./motifs/postgres/motif.yaml` |
| OCI registry | `motif: oci://ghcr.io/orkspace/patterns/motifs/postgres:v0.1.0` |
| Git URL | `motif: https://github.com/myorg/postgres-motif@main` |
| Orkestra Registry shorthand | `motif: postgres@v0.1.0` (resolved via `ORK_REGISTRY`) |

```yaml
imports:
  # Local — development and testing
  - motif: ./motifs/postgres/motif.yaml
    with:
      image: "postgres:16"

  # OCI — versioned, immutable, production-safe
  - motif: oci://ghcr.io/orkspace/patterns/motifs/postgres:v0.1.0
    oci: true
    with:
      image: "{{ .spec.postgresImage }}"
      volumeSize: "{{ .spec.storage }}"

  # Git
  - motif: https://github.com/myorg/postgres-motif@main
    with:
      image: "{{ .spec.postgresImage }}"
```

### `imports` field reference

| Field | Required | Description |
|-------|----------|-------------|
| `motif` | yes | File path, OCI reference, Git URL, or registry shorthand. |
| `oci` | no | `true` to pull via OCI/ORAS protocol. |
| `version` | no | Explicit version (tag, branch, SHA). Ignored when `@` shorthand is used in `motif`. Defaults to `latest` (OCI) or `main` (Git). |
| `auth` | no | Registry credentials. Same auth model as `imports.files` in a Komposer. |
| `with` | no | Input bindings. Values are Go templates evaluated in the CR reconcile context. |

---

## `with:` binding rules

`with:` maps Motif input names to Go template expressions evaluated against the CR being reconciled.

```yaml
with:
  # CR field reference
  image: "{{ .spec.postgresImage }}"

  # With default fallback
  volumeSize: "{{ .spec.storage | default \"10Gi\" }}"

  # Literal value — no template needed
  enableUI: "false"

  # Composed value
  passwordSecretName: "{{ .metadata.name }}-secrets"
```

- Required inputs not in `with:` → validation error at startup.
- Optional inputs not in `with:` → the Motif's `default` is used.
- `with:` values are evaluated at reconcile time, not at load time.

---

## Composing multiple Motifs

A CRD entry can import any number of Motifs. Each is independent — resources do not conflict when each uses `{{ .metadata.name }}` as a prefix.

```yaml
spec:
  crds:
    platform:
      apiTypes:
        group: platform.example.io
        version: v1
        kind: Platform
      imports:
        - motif: oci://ghcr.io/orkspace/patterns/motifs/postgres:v0.1.0
          oci: true
          with:
            image: "{{ .spec.database.image }}"
            volumeSize: "{{ .spec.database.storage }}"

        - motif: oci://ghcr.io/orkspace/patterns/motifs/redis:v0.1.0
          oci: true
          with:
            image: "{{ .spec.cache.image }}"
            volumeSize: "{{ .spec.cache.storage }}"

      # The Katalog's own resources alongside the imported Motifs
      operatorBox:
        deployments:
          - name: "{{ .metadata.name }}-api"
            image: "{{ .spec.apiImage }}"
            reconcile: true
```

Resources from all imports are merged with `operatorBox` in declaration order. The `operatorBox` block wins on name conflict.

---

## Motif nesting

A Motif can itself import other Motifs. A `postgres-with-backup` Motif could import the base `postgres` Motif and add a backup CronJob on top.

```yaml
# In postgres-with-backup/motif.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: postgres-with-backup

inputs:
  - name: image
    default: "postgres:latest"
  - name: backupSchedule
    default: "0 2 * * *"

imports:
  - motif: oci://ghcr.io/orkspace/patterns/motifs/postgres:v0.1.0
    with:
      image: "{{ .inputs.image }}"

resources:
  cronJobs:
    - name: "{{ .metadata.name }}-backup"
      schedule: "{{ .inputs.backupSchedule }}"
      image: "postgres:latest"
      command: ["pg_dump"]
```

---

## Real example — deployment-stack Katalog

The `deployment-stack` pattern in the Orkestra Registry is a Katalog that imports a single Motif and passes all CR spec fields through `with:`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: deployment-stack

spec:
  crds:
    application:
      apiTypes:
        group: apps.orkestra.io
        version: v1
        kind: Application
      operatorBox:
        reconcile:
          workers: 2
          resync: 1m
      imports:
        - motif: oci://ghcr.io/orkspace/patterns/motifs/deployment-stack:v0.1.0
          with:
            name: "{{ .metadata.name }}"
            namespace: "{{ .metadata.namespace }}"
            image: "{{ .spec.image }}"
            port: "{{ .spec.port }}"
            replicas: "{{ .spec.replicas }}"
            maxReplicas: "{{ .spec.maxReplicas }}"
            resourceProfile: "{{ .spec.resourceProfile }}"
            host: "{{ .spec.host }}"
```

Try it:

```bash
ork init --pack advanced
cd advanced/16-custom-resources/06-full-platform-composition
ork run -f katalog.yaml
kubectl apply -f cr-small.yaml
```

---

→ Previous: [02-resources.md](02-resources.md) | [Motif index](index.md)
