# Motif

!!! note "Motif is not a Kubernetes CRD"
    `kubectl apply` will not work. Orkestra kinds are consumed by the `ork` CLI and runtime — not by the Kubernetes API server. [Your CRD is enough](/blog/your-crd-is-enough/).

A Motif is the smallest reusable unit in Orkestra's composition model. It declares named inputs and contributes resource blocks to a Katalog CRD entry. It cannot run alone — it must be imported by a Katalog.

```text
Motif     — named inputs + resource blocks. One concern.
    ↓
Katalog   — operator declaration. Imports Motifs via imports:.
    ↓
Komposer  — platform declaration. Composes Katalogs.
    ↓
E2E       — declarative end-to-end test for a Katalog.
```

---

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif

metadata:
  name: postgres
  version: v0.1.0
  description: >
    PostgreSQL StatefulSet with persistent storage, headless Service, and pgAdmin.
  author: orkspace
  license: Apache-2.0
  tags:
    - database
    - statefulset

profiles:                              # optional → see profiles schema
  networkPolicies: [...]
  resourceQuotas: [...]
  resources: [...]
  probes: [...]
  containerSecurity: [...]
  podSecurity: [...]

notes:                                 # optional — user-defined template functions
  functions:
    - name: serviceHost
      expression: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"

inputs:
  - name: image
    description: PostgreSQL image
    default: "postgres:latest"

  - name: replicas
    description: Number of replicas
    default: "1"

  - name: volumeSize
    description: PVC storage size
    default: "10Gi"

resources:
  onCreate:
    secrets:
      - name: "{{ .metadata.name }}-creds"
        once: true
        data:
          password: "{{ randomAlphanumeric 16 }}"

  statefulSets:
    - name: "{{ .metadata.name }}-postgres"
      image: "{{ .inputs.image }}"
      replicas: "{{ .inputs.replicas }}"
      volumeClaimTemplates:
        - name: data
          storageSize: "{{ .inputs.volumeSize }}"
          mountPath: /var/lib/postgresql/data

  services:
    - name: "{{ .metadata.name }}-postgres"
      port: 5432

status:
  fields:
    - path: postgresReady
      value: "{{ allReplicasReady .children.statefulset }}"
    - path: connectionString
      value: "postgresql://{{ .metadata.name }}-postgres.{{ .metadata.namespace }}.svc.cluster.local:5432"
```

---

## How a Motif is used

A Katalog imports a Motif at the CRD entry level via `imports:`. The Katalog binds CR field values to Motif inputs using `with:`.

```yaml
# In a Katalog
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
```

At reconcile time, Orkestra:
1. Resolves and fetches the Motif from the declared source.
2. Evaluates each `with:` value as a Go template against the CR being reconciled.
3. Merges the Motif's resources into the CRD entry alongside the Katalog's own `operatorBox`.
4. Runs `onCreate` resources exactly once (on CR creation). All other blocks run on every reconcile.

---

## `metadata`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Motif identifier. Used as the registry artifact name. |
| `version` | no | Semver tag shown in `ork patterns`. |
| `description` | no | Short description shown in the registry UI. |
| `author` | no | Author or org name. |
| `license` | no | SPDX identifier (e.g. `Apache-2.0`). |
| `tags` | no | Free-form tags for registry discoverability. |

---

## Pattern directory structure

```text
postgres/
  motif.yaml      ← required — the Motif spec
  README.md       ← optional — shown in registry UI
  example/
    katalog.yaml  ← optional — example Katalog importing this Motif
```

`motif.yaml` is the only required file. The registry identifies the artifact kind from `kind: Motif` in the file.

---

## Try it

Example 16/06 is the capstone Motif example: a `Platform` CRD imports a Motif that provisions a `MessageQueue`, `ObjectStore`, and `SearchCluster` as child CRs — all from a single CR apply.

```bash
ork init --pack advanced
cd advanced/16-custom-resources/06-full-platform-composition
ork run -f katalog.yaml
```

Apply a small Platform CR:

```bash
kubectl apply -f cr-small.yaml
```

Orkestra resolves `platform-motif.yaml`, evaluates the `with:` bindings, and creates all three child CRs. Delete the Platform CR and all children are garbage-collected.

To browse the production-ready Motifs in the Orkestra Registry:

```bash
ork patterns --kind Motif
```

---

## `notes`

A Motif can declare user-defined template functions in a `notes.functions:` block. When the Katalog imports the Motif, its notes are merged into the Katalog's FuncMap alongside the Katalog's own notes and all built-ins.

```yaml
notes:
  functions:
    - name: serviceHost
      description: Cluster-local service hostname
      expression: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Function name. Called as `{{ noteName }}` in templates. Must be a valid Go identifier. |
| `expression` | yes | Go template expression. May call built-in notes and any other declared note. |
| `description` | no | Human-readable description. Shown in `ork validate --notes`. |

If the same note name appears in both the Katalog and an imported Motif, it is a hard error at load time.

→ Full reference: [Notes concept](../../../concepts/notes/index.md)

---

## `profiles`

A Motif can declare its own `profiles:` block. When a Katalog imports the Motif, its profiles are merged into the Katalog's registry and become available to all CRD entries.

Supported keys in a Motif are a subset of the Katalog's profiles — everything except `reconcile` (which is operator-level, not resource-level):

| Key | Class |
|-----|-------|
| `profiles.networkPolicies` | NetworkPolicy ingress/egress rules |
| `profiles.resourceQuotas` | Hard resource quota limits |
| `profiles.limitRanges` | Container and pod limit items |
| `profiles.pdb` | PodDisruptionBudget min/max settings |
| `profiles.rollingUpdate` | Deployment rolling update strategy |
| `profiles.resources` | Container CPU and memory requests/limits |
| `profiles.probes` | Probe timing parameters |
| `profiles.containerSecurity` | Container-level securityContext |
| `profiles.podSecurity` | Pod-level securityContext |

If the same profile name appears in the same class in both the Katalog and an imported Motif, it is a hard error at load time.

→ Full reference: [User-Defined Profiles](../../../concepts/profiles/10-user-defined-profiles.md)

## Where to go

| Page | Covers |
|------|--------|
| [01-inputs.md](01-inputs.md) | `inputs` — declaration, required/optional, defaults, type hints |
| [02-resources.md](02-resources.md) | `resources` — blocks, template context, `onCreate`, `status`, `admission` |
| [03-importing.md](03-importing.md) | Importing into a Katalog — `imports:`, `with:` bindings, multiple Motifs |
| [Notes concept](../../../concepts/notes/index.md) | User-defined notes — expression syntax, built-in reference, composition |

---

## Where to go next

- [Katalog schema](../02-katalog/01-top-level.md) — where `imports:` lives on the CRD entry
- [operatorBox](../02-katalog/04-operatorbox.md) — the Katalog's own resources, merged alongside Motif resources
- [Komposer schema](../03-komposer/index.md) — composing multiple Katalogs
- [Orkestra Registry](../../../orkestra-registry/index.md) — publishing and consuming Motifs
- [User-Defined Profiles](../../../concepts/profiles/10-user-defined-profiles.md) — declaring and referencing profiles
