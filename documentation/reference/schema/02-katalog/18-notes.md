# notes

User-defined template functions declared at the Katalog or Motif level.
Once registered, each function is callable by name in every `{{ }}` expression across the Katalog — status fields, resource names, `when:` conditions, mutation rules, and conversion paths.

```yaml
notes:
  functions:
    - name: fullImage
      description: Qualified image reference combining image and tag
      expression: '{{ .spec.image }}:{{ .spec.tag | default "latest" }}'

    - name: inBusinessHours
      expression: '{{ and weekday (timeInWindow "09:00" "18:00") }}'

    - name: statusLabel
      expression: "{{ if inBusinessHours }}Active{{ else }}Suspended{{ end }}"
```

## `notes.include`

Pull function definitions from an external file to keep the Katalog compact. The file contains a `functions:` key with the same list structure as inline definitions.

```yaml
notes:
  include: ./notes/shared.yaml   # relative to the katalog file
  functions:
    - name: localOverride        # appended after included functions
      expression: "{{ .spec.tier }}-{{ .metadata.name }}"
```

`notes/shared.yaml`:

```yaml
functions:
  - name: fullImage
    expression: '{{ .spec.image }}:{{ .spec.tag | default "latest" }}'
  - name: serviceHost
    expression: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

Included functions come first. Inline `functions:` append after. If the same name appears in both, the inline definition wins silently. The `include:` path is resolved relative to the katalog file's directory — the katalog can be run from any working directory. The field is cleared from the runtime bundle after expansion.

## `notes.functions`

Each entry defines one user-defined function.

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Function name. Called as `{{ noteName }}` in templates. Must be a valid Go identifier. |
| `expression` | yes | Go template string. May call built-in notes and other user-defined notes (order-independent). |
| `description` | no | Human-readable description. Shown in `ork notes -f katalog.yaml`. |
| `shadow` | no | Set `true` to acknowledge that this function intentionally overrides a built-in note. |

## Conflict rules

| Situation | Result |
|-----------|--------|
| Two `spec.imports` Motifs declare the same name | Hard error at startup |
| Inline `notes.functions` and a Motif import declare the same name | Local wins silently |
| Name matches a built-in note | Warning logged; `shadow: true` silences it |

## Distributing via Motifs

Declare functions in a Motif and import them at the Katalog level via `spec.imports`. Every CRD in the Katalog can then call those functions.

```yaml
# motifs/team-notes/motif.yaml
notes:
  functions:
    - name: fullImage
      expression: '{{ .spec.image }}:{{ .spec.tag | default "latest" }}'
    - name: appLabel
      expression: "{{ .metadata.namespace }}-{{ .metadata.name }}"
```

```yaml
# katalog.yaml
spec:
  imports:
    - motif: ./motifs/team-notes/motif.yaml
```

A Motif can also use `notes.include:` — the path resolves relative to the Motif file.

## Where to go next

- [Notes concept](../../../concepts/notes/index.md) — built-in reference, composition guide
- [Motif schema](../01-motif/index.md) — `notes:` on a Motif
- [Runtime context (.ork.*)](../../runtime-context/index.md) — process-level facts available in every expression
