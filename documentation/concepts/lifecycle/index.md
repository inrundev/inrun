# Lifecycle in Orkestra

An Orkestra operator is a Katalog — a YAML artifact, not a binary. That changes where lifecycle management lives. Maturity, compatibility, deprecation, and deletion protection are fields in the Katalog itself. They travel with the artifact from first push to final end-of-life, and are enforced by the same tooling that validates and runs it.

!!! tip "Lifecycle follows the artifact"
    The same model that bakes tests into the artifact at publish time is the model that governs maturity, upgrade, compatibility, deprecation, and deletion. There is no separate lifecycle system to install. You write patterns, and the lifecycle comes with them.

---

## The full arc

| Stage | How it works |
|-------|-------------|
| **Create** | A Katalog is YAML — no build pipeline, no image, no binary |
| **Validate** | `ork validate` — offline, schema, security posture, maturity and compatibility checks |
| **Test** | `ork simulate` (sub-second, no cluster) + `ork e2e` (real cluster) |
| **Distribute** | `ork push` — gates run, proof baked into OCI annotations, artifact ships |
| **Inspect** | `ork inspect` — proof visible before pulling, before importing |
| **Consume** | `ork pull` — artifact arrives with its proof |
| **Upgrade** | Version bump in the Komposer — same push pipeline, same gates, proof baked into the new artifact; never automatic |
| **Deprecate** | `lifecycle.deprecation` — author sees it at `ork push`; consumers at `ork inspect`, `ork pull`, `ork validate`; `ork run` blocks unless acknowledged via Komposer `lifecycle.accept.patterns` |
| **Delete** | `security.deletionProtection` — webhook blocks `kubectl delete` on CRs, CRDs, and Orkestra infrastructure |

---

## Creation

A Katalog declares a CRD and its complete operator behavior. There is no build step.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: platform
spec:
  crds:
    database:
      apiTypes:
        ...
    app:
      apiTypes:
        ...
      dependsOn:
        database: healthy
```

Complex Katalogs can import Motifs — reusable fragments for security posture, probes, retry policy, and RBAC. Motifs are themselves versioned OCI artifacts.

---

## Validation and testing

`ork validate` runs entirely offline. It checks schema, security posture, structural correctness, and — when `lifecycle:` is declared — maturity signals and compatibility ranges.

`ork simulate` runs the real reconciler against an in-memory cluster. It asserts which resources are created in which reconcile cycle. Sub-second, no Docker, runs in CI.

`ork e2e` spins up a kind cluster, deploys the operator, and verifies behavior against real scheduling and webhooks. This is the outer gate before publishing.

Both are declarative — YAML files describing expectations, not test scripts. The tests ship with the pattern, are reviewable, and cannot be silently skipped.

---

## Distribution

```bash
ork push database:v1.0.0 ./database/
```

`ork push` runs both gates automatically. If either fails, the push is blocked. When they pass, results are baked into OCI annotations on the artifact:

| Annotation | What it records |
|-----------|-----------------|
| `io.orkestra.simulate.status` | Passed / no assertions / skipped |
| `io.orkestra.simulate.assertions` | Count of `expect:` blocks |
| `io.orkestra.e2e.status` | Verified against a real cluster |
| `io.orkestra.e2e.assertions` | Total expectations |
| `io.orkestra.katalog.typed` | Whether a custom runtime is required |
| `io.orkestra.lifecycle.maturity` | Maturity level declared by the author |
| `io.orkestra.lifecycle.deprecated` | Whether the pattern is deprecated |
| `io.orkestra.lifecycle.deprecated.message` | Deprecation message |
| `io.orkestra.lifecycle.deprecated.migrated_to` | Migration target |
| `io.orkestra.lifecycle.deprecated.timeline_from` | Deprecation window open date |
| `io.orkestra.lifecycle.deprecated.timeline_to` | End-of-life date |
| `io.orkestra.lifecycle.compat.kubernetes` | Kubernetes version range |
| `io.orkestra.lifecycle.compat.orkestra` | Orkestra version range |

The artifact is self-describing. A consumer reads its proof before pulling.

---

## Consumption

```bash
ork inspect database:v1.0.0
```

`ork inspect` shows simulation status, e2e status, assertion counts, maturity level, compatibility ranges, and deprecation state — before pulling, before importing.

```bash
ork pull database:v1.0.0
```

`ork pull` downloads the artifact, caches it, and recursively pulls any referenced Motifs. The artifact arrives with its proof intact.

---

## Upgrade

Patterns are versioned with OCI tags. `database:v1.0.0` and `database:v1.1.0` are distinct artifacts with distinct proofs — not binary releases that need a separate controller to manage.

```bash
ork push database:v1.1.0 ./database/
```

The same push pipeline runs on every version. The new artifact is self-describing. `ork inspect database:v1.1.0` shows its proof before anything is pulled or imported.

The Komposer references the new version explicitly:

```yaml
imports:
  registry:
    - url: database:v1.1.0
```

Until you change that reference, nothing upgrades. There is no autosync, no watch loop, no version resolver that silently bumps the version when a newer tag appears. The decision to run `v1.1.0` is a line of YAML in a file that goes through code review.

---

## The `lifecycle:` block

`lifecycle:` is a top-level field on a Katalog, at the same level as `metadata`, `spec`, and `gateway`. It is the policy layer for the artifact — it changes what tooling does, not what the operator does at runtime.

```yaml
lifecycle:
  maturity: stable
  deprecation:
    message: "Use database-v2. Improved connection pooling and status reporting."
    migratedTo: "oci://ghcr.io/myorg/database-v2:v1.0.0"
    timeline:
      from: "2026-09-01"
      to:   "2027-03-01"
  compatibility:
    kubernetes: ">=1.31"
    orkestra: ">=0.7.14"
```

All sub-fields are optional. Maturity and compatibility are advisory — read by tooling at validate time. Deprecation is enforced: `ork run` and the Gateway refuse to start a deprecated Katalog unless a Komposer has explicitly acknowledged it.

---

## Maturity

`lifecycle.maturity` signals the stability level of the pattern. It affects how `ork validate` warns and how the pattern appears in registry listings.

| Value | Meaning | `ork validate` behaviour |
|-------|---------|--------------------------|
| `alpha` | Experimental, breaking changes expected | Warning: not recommended for production |
| `beta` | Stabilising, API mostly settled | Warning: lower severity |
| `stable` | Production-ready, semantic versioning applies | No warning |
| `deprecated` | Replaced or abandoned | Warning if `lifecycle.deprecation` is not also set (block is the primary signal) |

When `maturity` is omitted, tooling treats the pattern as `stable` if it has a version ≥ 1.0.0, otherwise `beta`.

---

## Deprecation

```yaml
lifecycle:
  maturity: deprecated
  deprecation:
    message: "Use database-v2. Improved connection pooling and status reporting."
    migratedTo: "oci://ghcr.io/myorg/database-v2:v1.0.0"
    timeline:
      from: "2026-09-01"   # deprecation window opens — warning with countdown
      to:   "2027-03-01"   # end of life
```

Deprecation is part of the artifact and surfaced at every touchpoint. The author sees it first: `ork push` shows the exact warning consumers will see before the artifact is uploaded. After that, `ork validate`, `ork inspect`, and `ork pull` all surface the same state-aware warning:

| Condition | Shown as |
|-----------|----------|
| No `timeline`, or `today < from` | ⚠ Deprecated |
| `from ≤ today < to` | ⚠ Deprecated · N days until EOL |
| `today ≥ to` | ✗ END OF LIFE |

The pattern remains in the registry — it is not deleted — but is clearly marked at every touchpoint.

**Runtime enforcement** — `ork run` refuses to start a deprecated or EOL Katalog when run directly. To run a deprecated Katalog, import it into a Komposer and acknowledge it there:

```yaml
lifecycle:
  accept:
    patterns:
      - name: database
        author: myorg
```

The acceptance lives on the Komposer — the consumer making a deliberate, reviewable decision — not on the Katalog itself.

---

## Compatibility

`lifecycle.compatibility` declares which Kubernetes and Orkestra versions this pattern is verified to work with. Both fields accept a semver range.

```yaml
lifecycle:
  compatibility:
    kubernetes: ">=1.31"
    orkestra: ">=0.7.14"
```

`ork validate` checks the declared range at validation time and warns when an import falls outside the declared bounds.

Accepted range syntax follows semver conventions:

```text
>=1.31           # at least 1.31
>=1.28, <1.33    # between 1.28 and 1.33 exclusive
^1.31            # >=1.31, <2.0
```

---

## Deletion protection

Deletion protection attaches a label to every managed CR and CRD, and registers a validating webhook that blocks any `kubectl delete` on labeled resources. When enabled, it protects both your CRs and Orkestra's own infrastructure.

```yaml
security:
  operatorBox:
    runtime:
      deletionProtection:
        enabled: true
```

Per-CRD overrides let you opt individual CRDs out of CR-level or CRD-level protection independently. See [Deletion Protection](../../security/04-deletion-protection.md) for the full configuration, strict mode, and gateway-only behavior.

---

