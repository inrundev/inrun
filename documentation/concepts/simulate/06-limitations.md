# Limitations

The fake cluster has no external connectivity and no real API server. Some operator features require a real cluster or a compiled user binary.

---

## Blocks that are inactive in simulation

| Block | Status | What you see |
|-------|--------|--------------|
| `reconcile.remote:` | **Skipped** — dispatches to an external endpoint at runtime; a note is printed and the CRD is omitted from the simulation | Use `ork e2e` to test against a live cluster |
| `external:` HTTP calls | Active by default — calls hit the real network; pass `--skip-external` to stub with empty 200 | Without `--skip-external`: real HTTP; with it: `external.*` fields are empty, note printed |
| `cross:` informer reads | Active when peer CRs are in the CR file | Include all sibling CRDs' CRs separated by `---` in the CR file. Each is seeded into a fake informer so `cross.*` fields populate. Without a peer CR, `cross.*` fields are empty and a note is printed. |
| `fromNamespace` / `toNamespaces` copies | **Automatically skipped** — no real API server to read from | A `note:` line is printed for each skipped resource before the first cycle. All other resources in the same phase proceed normally. Use `ork e2e` to verify the copy against a live cluster. |

The output still shows whether the declarative layer (templates, status fields, `once:`, `forEach:`) is correct given absent data.

---

## Standard binary fallback

When running from the standard `ork` binary (not a custom operator binary), `HookRegistry` and `ReconcilerRegistry` are empty for your custom types. Hook bodies and constructor reconcile loops do not execute. Simulate falls back to `Generic Reconciler` and runs the status layer only.

Build and use your own `ork` binary to get full hook and constructor simulation:

```bash
make registry && make build
ork simulate --cr cr.yaml
```

---

## What simulate cannot cover

| Feature | `--envtest` | `ork e2e` |
|---------|-------------|-----------|
| Real CRD schema enforcement | Yes | Yes |
| Status subresource (`.status` not patchable via normal patch) | Yes | Yes |
| Irregular plural resource names | Yes | Yes |
| Real watch stream delivery | Yes | Yes |
| Admission webhook validation/mutation | No | Yes |
| Real pod scheduling and readiness | No | Yes |
| Cross-namespace secret reads | No | Yes |
| Provider blocks (AWS, MongoDB) | No | Yes |

`--envtest` covers most API-server correctness concerns without a running cluster. Use `ork e2e` for admission webhooks, real pod lifecycle, and provider integrations.

→ See [Running with envtest](07-envtest.md) for setup and usage.

---

→ Back to: [ork simulate](index.md)
