# Writing Your First Motif

A **Motif** is a reusable building block. It declares named inputs and a set of Kubernetes resources. You write it once, publish it, and import it into any Katalog — with different inputs each time.

Motifs are not operators. They do not watch CRDs and they cannot run alone. A Motif is the _what_; a Katalog is the _when and who_.

---

## What you will build

A `redis` Motif that creates a Redis Deployment, a headless Service, and a credentials Secret. Any Katalog that needs Redis imports it with two lines.

---

## Step 1 — Create the Motif file

Create `motif.yaml`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif

metadata:
  name: redis
  version: v0.1.0
  description: Redis Deployment with headless Service and credentials Secret.
  author: platform-team

inputs:
  - name: image
    description: Redis image tag
    default: "redis:7-alpine"
  
  - name: replicas
    description: Number of replicas
    default: "1"

  - name: password
    description: Redis password (leave empty to disable auth)
    default: ""
  
  - name: rotateAfter
    description: How long before the password is rotated
    default: 30d

resources:
  onCreate:
    secrets:
      - name: "{{ .inputs.name }}-redis-creds"
        once: true
        rotateAfter: "{{ .inputs.rotateAfter }}"
        data:
          password: "{{ .inputs.password }}"

  deployments:
    - name: "{{ .inputs.name }}-redis"
      image: "{{ .inputs.image }}"
      replicas: "{{ .inputs.replicas }}"
      port: "6379"
      reconcile: true
  services:
    - name: "{{ .inputs.name }}-redis"
      port: "6379"
      targetPort: "6379"
      reconcile: true
status:
  fields:
    - path: redisReady
      value: "{{ replicasReady .children.deployment }}"
```

The template context is the CR being reconciled — `.inputs.name` is the CR's name, not the Motif's name. The Motif does not know which CRD will import it.

---

## Step 2 — Validate it

```bash
ork validate -f motif.yaml
```

`ork validate` checks:
- Required fields are present
- `inputs` names are unique and do not clash with reserved keys
- Template expressions in `resources` are syntactically valid
- `with:` bindings in any imports satisfy `required: true` inputs

No cluster is needed.

---

## Step 3 — Import the Motif into a Katalog

Create `katalog.yaml` alongside `motif.yaml`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: myapp

spec:
  crds:
    myapp:
      crdFile: ./crd.yaml
      crFiles:
        - ./cr.yaml

      imports:
        - motif: ./motif.yaml
          with:
            image: '{{ .spec.redisImage | default "redis:7-alpine" }}'
            replicas: "{{ .spec.replicas }}"
            password: "{{ randomBase64 32 }}"       # Orkestra random note

      operatorBox:
        deployments:
          - name: "{{ .metadata.name }}"
            image: "{{ .spec.image }}"
            replicas: "{{ .spec.replicas }}"
            reconcile: true
```

`imports` is where Motifs are bound. Each `with:` value is a Go template evaluated in the CR's reconcile context — the same context available in `operatorBox` templates.

---

## Step 4 — Run the Katalog

```bash
ork run
```

Orkestra loads the Katalog, expands the Motif import, and starts the operator. When a CR is applied:

1. The `redis-creds` Secret is created once (via `onCreate.secrets`)
2. The `<cr-name>-redis` Deployment and Service are reconciled every cycle
3. The `<cr-name>` Deployment from the Katalog's own `operatorBox` is also reconciled

The CR's status will show `redisReady: "true"` once the Redis Deployment is available.

---

## How inputs flow

```text
CR spec                              Motif
.spec.redisImage        →  with.image  →  inputs.image  →  {{ .inputs.image }}
.spec.redisPassword     →  with.password → inputs.password → {{ .inputs.password }}
```

- A `with:` value can be a Go template: `"{{ .spec.redisImage }}"`, a static string: `"redis:7-alpine"`, or empty string: `""`.
- Required inputs (no `default:`) that are not in `with:` are caught at startup by `ork run`, not at reconcile time.
- Optional inputs not in `with:` use the Motif's `default:`.

---

## Publishing to the registry

Once the Motif works locally, push it:

```bash
ork push ./<motif-dir> --registry oci://ghcr.io/myorg/patterns/redis
```

Other Katalogs can then import it by OCI reference:

```yaml
imports:
  - motif: oci://ghcr.io/myorg/patterns/redis:v0.1.0
    oci: true
    with:
      image: "{{ .spec.redisImage }}"
```

---

## Project layout

```text
redis/
  motif.yaml          # the Motif
  example/            # (optional)
    katalog.yaml      # example Katalog importing this Motif
    crd.yaml          # CRD definition
    cr.yaml           # CR definition
```

---

## Next steps

- **[Writing Your First Katalog](./03-writing-your-first-katalog.md)** — import this Motif into a full operator
- **[Motif schema reference](../reference/schema/01-motif/index.md)** — complete field reference
- **[Orkestra Registry](../orkestra-registry/01-motifs.md)** — browse published Motifs
