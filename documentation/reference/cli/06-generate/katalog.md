# ork generate katalog

Scaffold a starter `katalog.yaml` for a new operator.

```bash
ork generate katalog [flags]
```

Generates a `katalog.yaml` with sensible defaults, commented optional blocks, and `TODO` placeholders. All generated fields are valid YAML — no further tooling is required before editing.

---

## Flags

| Flag | Description |
|------|-------------|
| `--add-hook` | Typed mode: include a commented `hooks` declaration |
| `--add-constructor` | Typed mode: include a commented `constructor` declaration with `default: false` |
| `--typed` | Typed mode: include both `hooks` and `constructor` sections commented; prints a warning |
| `--add-security` | Include a security block (namespace and deletion protection) |
| `--add-notification` | Include a notification block with example team entries |
| `-o, --output <file>` | Output file path (default: `katalog.yaml`) |

---

## Reconcile Modes

Choose **at most one** reconcile mode flag. They are mutually exclusive.

| Flag | Mode | Description |
|------|------|-------------|
| *(none)* | Dynamic | Declarative templates (`onCreate`, `onReconcile`, `onDelete`). No Go code needed. |
| `--add-hook` | Typed | Commented `hooks` declaration. You write a `ReconcileHooks[*MyKind]` factory in Go. |
| `--add-constructor` | Typed | Commented `constructor` declaration with `default: false`. You own the full reconcile loop. |
| `--typed` | Typed | Both `hooks` and `constructor` sections commented. A warning is printed — you must uncomment exactly one before running. |

---

## Usage

Scaffold a dynamic-mode Katalog (default):

```bash
ork generate katalog
```

Typed mode with hooks:

```bash
ork generate katalog --add-hook -o database-katalog.yaml
```

Typed mode with a custom constructor:

```bash
ork generate katalog --add-constructor -o reconciler-katalog.yaml
```

Full typed mode (both sections commented, choose one):

```bash
ork generate katalog --typed
```

Add optional blocks:

```bash
ork generate katalog --add-security --add-notification
```

Combine mode and optional flags:

```bash
ork generate katalog --add-hook --add-security -o ops-katalog.yaml
```

---

## Behavior

- Validates flag combinations before writing any file. Mutually exclusive flags produce a clear error.
- Writes the scaffold to `katalog.yaml` (or `--output` path).
- The output is pure YAML — no template syntax leaks into the generated file.
- When `--typed` is used, a warning is printed to stderr reminding you to uncomment exactly one section.

---

## Optional Blocks

Optional flags may be combined freely with any reconcile mode.

| Flag | Block added |
|------|-------------|
| `--add-security` | `security.namespaceProtection` and `security.deletionProtection` |
| `--add-notification` | `notification.teams` with example Slack webhook entries |

---

## Notes

- Replace every `TODO` placeholder before running `ork run`.
- The generated file is idempotent — re-running overwrites the output file.
