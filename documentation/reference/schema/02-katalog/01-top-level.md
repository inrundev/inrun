# Katalog

A Katalog declares one or more CRDs and defines how Orkestra manages them.
It is the **unit of operator definition**.

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1   # required
kind: Katalog                          # required

metadata:
  name: my-operator                    # required
  description: string                  # optional

profiles:                              # optional → see profiles schema
  reconcile: [...]
  networkPolicies: [...]
  resourceQuotas: [...]
  limitRanges: [...]
  pdb: [...]
  rollingUpdate: [...]
  resources: [...]
  probes: [...]
  containerSecurity: [...]
  podSecurity: [...]

notes:                                 # optional — user-defined template functions
  functions:
    - name: fullImage
      description: string
      expression: '{{ .spec.image }}:{{ .spec.tag | default "latest" }}'

spec:
  finalizers:                          # optional — applied to every CRD
    - platform.example.io/cleanup

  crds:                                # required — map of CRD entries by name
    <name>:                            # ← map key is the CRD name
      ...                              # → see crd-entry.md

security:                              # optional → see katalog-security.md
  ...

notification:                          # optional → see katalog-notification.md
  ...

providers:                             # optional → see katalog-providers.md
  - ...
```

## `metadata`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unique identifier. Written as the `managed-by` annotation on all CRs. |
| `description` | no | Shown in the `/katalog` API response. |

## `spec.finalizers`

Katalog-level finalizers applied to every CRD in this Katalog.
Override per-CRD via [`operatorBox.finalizers`](04-operatorbox.md).

## `spec.crds`

A **map** — the key is the CRD name, the value is a `CRDEntry`.

```yaml
spec:
  crds:
    database:          # ← CRD name
      enabled: true
      apiTypes:
        group: apps.example.io
        version: v1alpha1
        kind: Database
        plural: databases
```

→ Full field reference: [crd-entry.md](02-crd-entry.md)

## `notes`

User-defined template functions, available in every `{{ }}` expression in this Katalog — status fields, resource names, `when:` conditions, and anywhere else a template is evaluated.

```yaml
notes:
  functions:
    - name: fullImage
      description: Qualified image reference combining image and tag
      expression: "{{ .spec.image }}:{{ .spec.tag | default \"latest\" }}"

    - name: inBusinessHours
      expression: '{{ and weekday (timeInWindow "09:00" "18:00") }}'

    - name: statusLabel
      expression: "{{ if inBusinessHours }}Active{{ else }}Suspended{{ end }}"
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Function name. Called as `{{ noteName }}` in templates. Must be a valid Go identifier. |
| `expression` | yes | Go template expression. May call built-in notes and other user-defined notes. |
| `description` | no | Human-readable description. Shown in `ork validate --notes`. |

Notes are pure: same input → same output. They may call any built-in note and any other user-defined note declared in the same `notes.functions:` block (order-independent).

→ Full reference: [Notes concept](../../../concepts/notes/index.md)

---

## `profiles`

Named profile definitions shared across all CRD entries in this Katalog. Profiles are resolved by name at reconcile time — user-defined profiles take precedence over built-ins.

| Key | Class |
|-----|-------|
| `profiles.reconciler` | Reconciler tuning (workers, resync, queue depth) |
| `profiles.networkPolicies` | NetworkPolicy ingress/egress rules |
| `profiles.resourceQuotas` | Hard resource quota limits |
| `profiles.limitRanges` | Container and pod limit items |
| `profiles.pdb` | PodDisruptionBudget min/max settings |
| `profiles.rollingUpdate` | Deployment rolling update strategy |
| `profiles.resources` | Container CPU and memory requests/limits |
| `profiles.probes` | Probe timing parameters |
| `profiles.containerSecurity` | Container-level securityContext |
| `profiles.podSecurity` | Pod-level securityContext |

→ Full reference: [User-Defined Profiles](../../../concepts/profiles/10-user-defined-profiles.md)

## Where to go next

- [crd-entry.md](02-crd-entry.md)
- [katalog-security.md](10-katalog-security.md)
- [katalog-notification.md](11-katalog-notification.md)
- [katalog-providers.md](12-katalog-providers.md)
- [komposer.md](../03-komposer/index.md) — compose multiple Katalogs
- [User-Defined Profiles](../../../concepts/profiles/10-user-defined-profiles.md) — declaring and referencing profiles
- [Notes concept](../../../concepts/notes/index.md) — user-defined notes, built-in reference, composition
