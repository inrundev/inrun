# Declarative Unit Testing

`ork simulate` runs the operator reconcile loop against a fake in-memory cluster. No Kubernetes, no `kubectl`, no network — results in milliseconds.

---

## Pages

| Page | What it covers |
|------|----------------|
| [How it works](01-how-it-works.md) | The fake cluster model, same reconciler, steady state detection |
| [simulate.yaml](02-simulate-kind.md) | The recommended entry point — schema, `expect:`, assert mode, validation |
| [Running simulate](03-running.md) | All invocation forms: `--cr`, `-f simulate.yaml`, `./...`, flags |
| [Hooks and constructors](04-hooks-and-constructors.md) | Custom binary, registry wiring, what the standard binary shows |
| [Aggregator mode](05-aggregator.md) | `./...` discovery and aggregator simulate files |
| [Limitations](06-limitations.md) | What simulate cannot cover and what to use instead |
| [Running with envtest](07-envtest.md) | Real kube-apiserver behind `--envtest` — schema enforcement, watch streams, no cluster |

---

## The key property

`ork simulate` does not approximate what the reconciler does. It *is* the reconciler — the same `Generic Reconciler` that runs in production, wired to a fake in-memory Kubernetes store. Template expressions, `when:` conditions, `onCreate`/`onReconcile` order, and status propagation all execute identically.

Use simulate as the fast inner loop while writing an operator. Use `ork e2e` as the outer gate before pushing.

| | `ork simulate` | `ork simulate --envtest` | `ork e2e` |
|---|---|---|---|
| Requires cluster | No | No | Yes |
| Runs real reconciler | Yes | Yes | Yes |
| Real CRD schema enforcement | No | Yes | Yes |
| Real watch streams | No | Yes | Yes |
| Tests webhooks | No | No | Yes |
| Tests external calls | Yes | Yes | Yes |
| Speed | Milliseconds | ~3–5s | Minutes |
| Best for | Template correctness | API-server correctness | System correctness |

---

## Where to go next

- **[How it works](01-how-it-works.md)** — the fake cluster model, same reconciler, steady state detection
- **[simulate.yaml](02-simulate-kind.md)** — schema, `expect:`, assert mode, op-print mode, validation
- **[Running simulate](03-running.md)** — all invocation forms, `--dev-server`, `./...` discovery, flag reference

→ See also: [`ork simulate` CLI reference](../../reference/cli/05-simulate.md) | [Declarative Integration Testing](../envtest/index.md)