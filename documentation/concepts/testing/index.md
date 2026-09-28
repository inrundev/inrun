# Testing in Orkestra

Testing is built into every layer of Orkestra — not added on top. Intent delivery, admission rules, webhook intake, and reconciliation each have a dedicated tool that runs locally, without a cluster, in milliseconds. `ork e2e` is the final gate for what only Kubernetes can test.

---

## The full picture

| Tool | What it tests | Needs cluster | Speed |
|------|--------------|---------------|-------|
| `ork serve play` | Intent delivery chain — target resolution, token check, CR construction, provenance, admission, response | No | ms |
| `ork gate` | Admission rules against a pre-built CR | No | ms |
| `ork gate run` | Full local gateway for `ork serve apply` round-trips | No | ms |
| `ork webhook play` | Webhook intake — branch filters, watch patterns, content fetch, same delivery chain | No | ms |
| `ork simulate` | Reconciliation — full operatorBox against an in-memory cluster | No | ms |
| `ork simulate --envtest` | Same reconciler against a real kube-apiserver + etcd | No | ~3–5s |
| `ork e2e` | System behaviour — real cluster, real scheduling, real webhooks | Yes | mins |

Six tools. Two clusters of concerns separated by a single line.

---

## The cluster line

Everything above the line tests a different layer of the same system. Nothing connects to a cluster. Nothing waits on pod scheduling or image pulls. These tools finish in milliseconds because they test the logic of the system, not its interaction with Kubernetes.

Everything below the line — just `ork e2e` — tests what only Kubernetes can test: real admission webhooks with TLS, actual resource creation and garbage collection, scheduler decisions, pod readiness, finalizer sequencing.

The goal is to push as much verification as possible above the line. `ork e2e` is the gate before you push to a registry, not the gate before you write the next line.

---

## The delivery layer

The Gateway has three entry points: the CLI (`ork serve apply`), the Webhook intake, and eventually a Control Center form. All three converge on the same six-stage chain: target resolution → token check → CR construction → provenance stamping → admission validation → response payload.

**`ork serve play`** runs that full chain from a flat intent file. No gateway process, no cluster. You give it an intent and a token; it prints the CR it would build, the admission result, and the response payload the caller would receive. A field that fails target resolution fails here with the same message it would fail in production.

```bash
ork serve play --token ci-pipeline
ork serve play --token ci-pipeline --target preview   # test alias restrictions
ork serve play --token ci-pipeline --simulate          # continue into reconciliation
```

**`ork gate`** runs just the admission stage, but from a CR you provide directly — not from an intent file. Use it to test admission rules in isolation, reproduce a webhook denial, or check a CR from another source (kubectl, a GitOps controller) before applying it.

```bash
ork gate -f katalog.yaml --cr cr-staging.yaml
```

**`ork gate run`** starts the full gateway process locally — HTTP only, no TLS — so you can run `ork serve apply` against it for end-to-end delivery testing without a deployed cluster or a Helm chart.

```bash
# Terminal 1
ork gate run -f katalog.yaml

# Terminal 2
ork serve apply -f intent.yaml -t my-token
```

**`ork webhook play`** tests the webhook intake path. Webhook sources have their own logic before the delivery chain starts — branch filters, watch pattern matching, content fetching, command parsing. `ork serve play` can't test that. `ork webhook play` runs the full intake path, including source-specific parsing, from a simulated payload.

```bash
ork webhook play --webhook payments-repo \
  --event push-event.json \
  --fetch services/payments/intent.yaml=./intent.yaml
```

→ [Gate and local gateway](01-gate.md) | [Webhook play](02-webhook-play.md) | [Local intent testing](../self-service/06-local-intent-testing.md)

---

## The reconciler layer

**`ork simulate`** runs the real reconciler — not a mock, the same `Generic Reconciler` that runs in production — against an in-memory Kubernetes store. Template expressions, `when:` conditions, `onCreate`/`onReconcile` order, status propagation, all cycles. Sub-second.

```yaml
# simulate.yaml
spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  cycles: 5            # default 10 when omitted
  skipExternal: false  # stub external: HTTP calls (default: hit real network)

  expect:
    steady: true       # reconciler must stabilise within cycles
    noErrors: true     # any reconcile error fails the run
    ops:
      - cycle: 1
        verb: create
        resource: statefulsets
      - cycle: 1
        verb: create
        resource: cronjobs
        name: my-db-backup   # optional: assert a specific resource name
```

```bash
ork simulate -f simulate.yaml
```

**`ork simulate --envtest`** runs the same loop against a real `kube-apiserver` + `etcd`, spun up locally in ~3 seconds. No cluster, no deployed operator. It fills the gap between the fake clients in `ork simulate` and the full cluster in `ork e2e`: CRD schema enforcement, real watch streams, proper REST mapper.

```bash
ork simulate -f simulate.yaml --envtest
```

→ [Declarative unit testing](../simulate/index.md) | [Declarative integration testing](../envtest/index.md)

---

## The system layer

**`ork e2e`** verifies the operator against a real Kubernetes cluster. It provisions a kind cluster, installs Orkestra, applies your CRDs and CRs, waits on checkpoints, and verifies cleanup. This is the only layer that tests what cannot be tested locally: real admission webhooks with TLS, actual pod scheduling and image pulls, finalizer sequencing, controller-runtime leader election.

```bash
ork e2e -f e2e.yaml
```

The `e2e.yaml` declares what to apply, what checkpoints must pass, and in what order. The same file runs locally and in CI against the same kind cluster.

→ [Declarative end-to-end testing](../e2e/index.md)

---

## Composing the full chain

The tools compose. Once the basic intent file and simulate.yaml are written, the full local path from webhook push to reconciled child resources runs in one command:

```bash
# 1. Test the delivery chain from a CI intent
ork serve play --token ci-pipeline

# 2. Test admission rules against a pre-built CR
ork gate --cr .ork/cr.yaml

# 3. Test delivery into reconciliation
ork serve play --token ci-pipeline --simulate simulate.yaml

# 4. Test the webhook intake path into reconciliation
ork webhook play --webhook payments-repo \
  --event push-event.json \
  --fetch services/payments/intent.yaml=./intent.yaml \
  --simulate simulate.yaml

# 5. Test against a real cluster
ork e2e -f e2e.yaml
```

Steps 1–4 run in milliseconds. Step 5 runs in minutes. Steps 1–4 are the inner loop during development; step 5 is the gate before `ork push`.

By the time step 5 runs, you have already verified: intent routing, token scoping, CR construction, field translation, admission rules, reconciler output, child resource shape, status propagation, and webhook intake for every source you have configured. What `ork e2e` adds is the one thing none of the others can give you: Kubernetes itself.


## Where to go next

- [Gate and local gateway](01-gate.md) — `ork gate` and `ork gate run`
- [Webhook play](02-webhook-play.md) — `ork webhook play`
- [Declarative unit testing](../simulate/index.md) — `ork simulate`
- [Declarative integration testing](../envtest/index.md) — `ork simulate --envtest`
- [Declarative end-to-end testing](../e2e/index.md) — `ork e2e`
- [Local intent testing](../self-service/06-local-intent-testing.md) — `ork serve play` in depth
