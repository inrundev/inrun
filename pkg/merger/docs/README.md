# Merger — Developer Documentation

This directory explains how the `pkg/merger` package resolves and merges Katalog and Komposer YAML files into the unified CRD map the operator boots from.

## Documents

| File | What it covers |
|------|----------------|
| [01-architecture.md](01-architecture.md) | The full merge pipeline from entry-point files to a ready `*Merger` |
| [02-kinds.md](02-kinds.md) | Katalog vs Komposer — what each kind can and cannot declare |
| [03-imports.md](03-imports.md) | Import types (file, Helm, registry) — resolution order and auth |
| [04-deduplication.md](04-deduplication.md) | How duplicate CRD names are detected and why they are errors |
| [05-top-level-accumulation.md](05-top-level-accumulation.md) | How security is merged across imports |
| [06-protocol.md](06-protocol.md) | The v1 wire protocol: required fields, apiVersion enforcement, format rules |

Read them in order the first time. For the v1 contract (what external patterns must look like), read [06-protocol.md](06-protocol.md).
