# How it works

`ork simulate` builds a fake in-memory Kubernetes cluster, seeds it with your CR, then drives the reconciler through N cycles. Each cycle records every create, update, patch, and delete that hit the fake store. When two consecutive cycles produce identical operations, steady state is declared.

---

## The fake cluster

The fake cluster is a thin wrapper around `k8s.io/client-go/dynamic/fake` and `k8s.io/client-go/kubernetes/fake`. It intercepts all Kubernetes API calls made by the reconciler and stores the results in memory. No real API server is involved.

At startup, the cluster contains only the CR you provide. The informer cache is backed by a static indexer containing that CR. Resources created by the reconciler in cycle 1 are immediately available in subsequent cycles — but there is no scheduler, no kubelet, and no controller-manager. Deployments are marked ready immediately so status conditions can propagate.

---

## The reconciler

`ork simulate` by default runs the same `Generic Reconciler` that runs in production. When your operator binary registers a custom constructor or hooks, simulate runs that code instead — still the actual production reconciler, not a mock.

This means simulation catches things that static analysis cannot:

- Template expression errors (bad field references, missing map keys)
- Missing `when:` guards that cause unintended resource creation on the first reconcile
- Status fields that reference child resources not yet created
- `onReconcile` blocks that diverge from `onCreate` when they should produce identical output
- `once:` guards that fire every cycle because the resource was never found

---

## Cycle output

```text
Simulating website/my-site

  Cycle 1:
    + deployments/my-site
    + services/my-site
    ~ status/my-site

  Cycle 2:
    ~ status/my-site

  (cycles 3–10: identical)

  ✓ Steady state at cycle 3 in 189ms
```

`+` created. `~` updated/patched. `-` deleted.

`status/...` appearing on every cycle is expected — status fields are re-evaluated each reconcile. Everything else should reach steady state by cycle 2 or 3. A resource that keeps appearing as `+` after cycle 1 is being re-created instead of found stable, usually because a template expression produces a different value on each evaluation.

Identical consecutive cycles are collapsed. Steady state is noted but does not stop the run — all `--cycles` cycles execute.

---

## The development loop

1. Write a Katalog and a minimal CR
2. Run `ork simulate` — verify the right resources appear in cycle 1
3. Check that cycle 2 shows only `status/...`
4. Adjust `when:` conditions, field references, or template expressions as needed
5. When simulation is clean, run against a real cluster with `ork e2e` and then `ork run`.
