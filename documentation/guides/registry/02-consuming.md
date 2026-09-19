# Consuming Patterns

Before writing a single YAML file, check whether the pattern you need already exists in the registry. The consumption flow is: **browse → inspect → pull → import**.

---

## Browse: ork patterns

```bash
ork patterns
```

```text
Orkestra Registry
─────────────────────────────────────────────────────────────────────
NAME              LATEST  KIND     E2E  TAGS                    DESCRIPTION
postgres          v1.0.0  Katalog  ✓    postgres, database,...  Declarative PostgreSQL...
redis             v1.0.0  Katalog  ✓    redis, cache,...        Declarative Redis...
web-service       v1.1.0  Motif    -    web, deployment,...     Deployment + Service +...
data-store        v1.0.0  Motif    -    statefulset, storage    StatefulSet + PVC + cr...
```

The `E2E` column shows whether the published artifact carries a verified E2E result. `-` for Motifs — Motifs have no E2E gate (they have no CRD to verify against; the proof lives in the Katalog that imports them).

Filter by kind or tag:

```bash
ork patterns --katalogs           # katalogs only
ork patterns --motifs             # motifs only
ork patterns --tag database       # any pattern tagged "database"
```

Browse a specific registry (not the default):

```bash
ork patterns oci://ghcr.io/myorg/katalogs
```

---

## Inspect: read the quality signal before pulling

`ork inspect` fetches metadata from the registry without downloading any files. Use it to read simulate and E2E status, digest, and the exact import snippet — before committing to a pull.

```bash
ork inspect postgres:v1.0.0
```

```text
postgres:v1.0.0
  Registry:    ghcr.io/orkspace/orkestra-registry/patterns/katalogs
  Kind:        Katalog
  Digest:      sha256:a3f1c8d20e4b7f9c1234567890abcdef...
  Pushed:      2026-05-24T10:00:00Z
  Size:        14.2 KB

  Description: Declarative PostgreSQL deployment operator

  Tags:        postgres, database, statefulset
  Author:      orkspace
  License:     Apache-2.0
  Simulate:    ✓ Verified · 3ms · tested 2 days ago
  E2E:         ✓ Verified · 48s · tested 2 days ago

  Files:
    katalog.yaml    5.1 KB
    crd.yaml        3.2 KB
    cr.yaml         512 B
    simulate.yaml   1.1 KB
    e2e.yaml        2.4 KB
    README.md       4.8 KB

To pull:
  ork pull postgres:v1.0.0

To import:
  imports:
    registry:
      - oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
```

The quality signals are baked into the OCI artifact as annotations. `ork inspect` reads them without downloading the layer content — fast and zero-cost.

For a motif:

```bash
ork inspect web-service:v1.1.0 --motif
```

View a specific file without pulling the full artifact:

```bash
ork inspect postgres:v1.0.0 --view simulate.yaml
ork inspect postgres:v1.0.0 --view katalog.yaml,cr.yaml
```

This lets you read the simulate assertions — the exact behavioral proof the author published — before deciding to import.

---

## Pull: cache locally

```bash
ork pull postgres:v1.0.0
```

Downloaded to `~/.orkestra/registry/<registry>/<name>/<version>/`. The CLI resolves the short name against `ORK_REGISTRY`.

Pull a motif:

```bash
ork pull web-service:v1.1.0 --motif
```

Pull all imports declared in a file:

```bash
ork pull -f komposer.yaml      # pulls every registry import in komposer.yaml
ork pull -f katalog.yaml       # pulls OCI motif imports
```

Extract to a local directory (for inspection or local editing):

```bash
ork pull postgres:v1.0.0 -o ./local-postgres
```

Force a re-download even if the version is already cached:

```bash
ork pull postgres:v1.0.0 --refresh
```

---

## Import: use the pattern in a Komposer

The "To import:" snippet from `ork inspect` is the exact YAML to add to your Komposer:

```yaml
# komposer.yaml
imports:
  registry:
    - oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
```

Override any CRD-level setting without modifying the upstream pattern:

```yaml
imports:
  registry:
    - oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0

spec:
  crds:
    postgres:
      operatorBox:
        reconcile:
          workers: 8          # override the upstream default
          resync: 30s
```

Your inline `spec.crds` always wins over imported values. This is how platform teams customize community patterns for their environment.

---

## Re-run the simulate assertions yourself

The artifact contains `simulate.yaml`. You can pull it and run the same assertions the author ran before publishing:

```bash
ork pull postgres:v1.0.0 -o ./postgres-pattern
cd ./postgres-pattern
ork simulate
# ✓ 3/3 assertions passed
```

This is the trust model: the author proved these assertions before publishing. You can verify the proof applies to your environment too — no cluster needed.

---

## Try it

```bash
ork init --pack registry-guide
cd 00-consume
cat komposer.yaml          # imports postgres from the official registry
ork inspect postgres:v1.0.0
ork pull -f komposer.yaml
```

→ Next: [Publishing Motifs](03-publishing-motifs.md)
