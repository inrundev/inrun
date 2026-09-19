# include:

`include:` splits one logical unit across multiple files. The Katalog, Motif, E2E, or Simulate and its include files are one thing — same directory, same version, same lifecycle. After expansion the `include:` field is cleared and the runtime sees a single merged declaration.

It is authoring ergonomics, not reuse. When a block grows long, split it. No artifact, no version, no distribution.

`include:` is supported across all nine major blocks. It was originally available only in `e2e expect:`; it now covers every block that can grow long — admission, status, notes, profiles, and simulate ops.

A natural extension is to organise by **domain** rather than by block type — putting every concern for a given resource group (assertions, admission, status) in one folder so a domain-scoped change touches only that folder.

---

## Supported blocks

Every block that can grow long supports `include:`. The include file uses the same key as the inline block — the content format is identical.

| Block | `include:` field | File key | Order |
|-------|-----------------|----------|-------|
| `serve.fields` | `serve.include` | `fields:` | included first, inline wins on key conflict |
| `validation.rules` | `validation.include` | `rules:` | included first, inline appends after |
| `mutation.rules` | `mutation.include` | `rules:` | included first, inline appends after |
| `conversion.paths` | `conversion.include` | `paths:` | included first, inline appends after |
| `status.fields` | `status.include` | `fields:` | included first, inline appends after |
| `notes.functions` | `notes.include` | `functions:` | included first, inline appends after |
| `profiles` | `profiles.include` | `profiles:` | included first per sub-list, inline appends after |
| `expect` (E2E) | `expect[].include` | `expect:` | expanded in place at the position of the include entry |
| `expect.ops` (Simulate) | `ops[].include` | `ops:` | expanded in place at the position of the include entry |

---

## File structure

```text
my-operator/
  katalog.yaml
  admission/
    apprequest.yaml       ← validation rules
    appdefaults.yaml      ← mutation rules
  status/
    apprequest.yaml    ← status fields
  notes/
    shared.yaml        ← shared notes functions
```

`katalog.yaml` references each file by relative path:

```yaml
spec:
  crds:
    apprequest:
      admission:
        validation:
          include: ./admission/apprequest.yaml
          rules:
            - field: spec.tier           # inline — appended after included rules
              operator: in
              value: "free,pro,enterprise"
        mutation:
          include: ./admission/appdefaults.yaml
      operatorBox:
        emit:
          status:
            include: ./status/apprequest.yaml
notes:
  include: ./notes/shared.yaml
  functions:
    - name: localNote                  # inline — appended after included functions
      expression: "{{ .spec.tier }}-{{ .metadata.name }}"
```

Each include file:

```yaml
# admission/apprequest.yaml
rules:
  - field: spec.team
    operator: exists
    message: "spec.team is required"
    action: deny
  - field: spec.environment
    operator: in
    value: "staging,production"
    action: deny
```

```yaml
# notes/shared.yaml
functions:
  - name: fullImage
    expression: '{{ .spec.image }}:{{ .spec.tag | default "latest" }}'
  - name: serviceHost
    expression: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

```yaml
# profiles/shared.yaml
profiles:
  resources:
    - name: api-worker
      requests:
        cpu: "500m"
        memory: "256Mi"
```

---

## Motifs support include: too

A local Motif can use `include:` in the same blocks — status, admission, notes, profiles. Paths resolve relative to the Motif file, not the importing Katalog.

```yaml
# motifs/postgres/motif.yaml
status:
  include: ./status/postgres.yaml

notes:
  include: ./notes/postgres.yaml
```

OCI and Git Motifs are pre-packaged — their `include:` files have already been expanded before publishing.

---

## The E2E case

`expect:` in an E2E file supports `include:` at the checkpoint level. An include entry is expanded in place, so position in the list determines run order:

```yaml
# e2e.yaml
expect:
  - include: ./e2e/01-infra-ready.yaml    # expands here — runs first
  - name: Operator-specific check
    after: cr-applied
    timeout: 30s
    resources:
      - kind: MyOperator
        name: my-resource
  - include: ./e2e/02-cleanup.yaml        # expands here — runs last
```

By convention, include files live in an `e2e/` subdirectory alongside the `e2e.yaml`. Each file uses `expect:` as its root key — the same key as the inline block:

```yaml
# e2e/01-infra-ready.yaml
expect:
  - name: CRD registered
    after: cr-applied
    timeout: 30s
    kubectl:
      get:
        - kind: CustomResourceDefinition
          name: myoperators.example.com
          field: status.conditions[0].type
          equals: Established
```

`ork validate` shows the fully-expanded checkpoint list in run order. Nested includes (an included file that itself includes another) are not supported.

→ [E2E expect: reference](../../reference/schema/04-e2e/03-expect.md)

---

## The Simulate case

`ops:` in a Simulate file supports `include:` at the op level. An include entry is expanded in place, preserving position and grouping:

```yaml
# simulate.yaml
spec:
  expect:
    steady: true
    noErrors: true
    ops:
      - include: ./ops/01-namespaced.yaml
      - include: ./ops/02-rbac.yaml
      - include: ./ops/03-workloads.yaml
```

By convention, include files live in an `ops/` subdirectory alongside `simulate.yaml`. Each file uses `ops:` as its root key:

```yaml
# ops/01-namespaced.yaml
ops:
  - cycle: 1
    verb: create
    resource: namespaces
    name: acme-dev
  - cycle: 1
    verb: create
    resource: namespaces
    name: acme-staging
```

This is useful in fixtures with large `ops:` lists — such as a `forEach` fixture that expands two dozen resource types across multiple namespaces. Splitting by motif or resource group keeps each file focused.

`ork simulate` shows the fully-expanded op list before asserting. Nested includes (an included file that itself includes another) are not supported.

→ [Simulate ops reference](../../reference/schema/05-simulate/index.md)

---

## Domain layout pattern

Instead of a flat set of include files, you can organise by **domain** — one folder per resource group, containing every concern for that group. A change to workload behaviour touches `includes/workloads/` only; the PR diff is scoped and the commit message writes itself.

```text
my-operator/
  katalog.yaml
  e2e.yaml
  simulate.yaml
  includes/
    workloads/
      admission/
        validation.yaml     ← rules:  (separate file — same key as mutation)
        mutation.yaml       ← rules:
      ops.yaml              ← ops:    (simulate assertions)
      expect.yaml           ← expect: (e2e checkpoints)
      status.yaml           ← fields:
    rbac/
      admission/
        validation.yaml
        mutation.yaml
      ops.yaml
      expect.yaml
    storage/
      ops.yaml
      expect.yaml
    shared/
      notes.yaml            ← functions: (cross-CRD, referenced by multiple katalogs)
      profiles.yaml         ← profiles:  (cross-CRD)
```

`admission/` is a folder because validation and mutation both use `rules:` as their root key — they cannot share a single file.

`notes` and `profiles` sit in `shared/` because they are typically cross-cutting: the same note function or resource profile applies to multiple CRDs. They are the exception, not the rule.

`katalog.yaml` wires everything together:

```yaml
spec:
  crds:
    deployment:
      admission:
        validation:
          include: ./includes/workloads/admission/validation.yaml
        mutation:
          include: ./includes/workloads/admission/mutation.yaml
      operatorBox:
        emit:
          status:
            include: ./includes/workloads/status.yaml
    rolebinding:
      admission:
        validation:
          include: ./includes/rbac/admission/validation.yaml
notes:
  include: ./includes/shared/notes.yaml

profiles:
  include: ./includes/shared/profiles.yaml
```

`simulate.yaml` and `e2e.yaml` reference the same domain folders:

```yaml
# simulate.yaml
ops:
  - include: ./includes/workloads/ops.yaml
  - include: ./includes/rbac/ops.yaml
  - include: ./includes/storage/ops.yaml

# e2e.yaml
expect:
  - include: ./includes/workloads/expect.yaml
  - include: ./includes/rbac/expect.yaml
  - include: ./includes/storage/expect.yaml
```

---

## imports: vs include: at a glance

| | `imports:` | `include:` |
|-|-----------|-----------|
| Unit boundary | Crosses unit boundaries | Within one unit |
| Versioning | Independent version per artifact | Same version as the declaring file |
| Distribution | OCI, Git, Helm, local file | Local file only |
| Lifecycle | Updated independently | Updated with the declaring file |
| Runtime visibility | Merged at load time, visible in bundle | Cleared at load time, not in bundle |
| Primary use | Share across operators and teams | Split large blocks, organise E2E checkpoints and Simulate ops |

→ Back: [imports:](01-imports.md) | [Composition overview](index.md)
