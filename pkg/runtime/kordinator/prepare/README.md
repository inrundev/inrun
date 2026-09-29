# prepare

Builds the `domain.PreparedRequest` for a single reconcile cycle. Called by
Kordinator after the reconcile gate passes, before the reconciler is invoked.
Also called by `simulate` before each cycle — both the in-memory and envtest
modes — so every simulation path exercises the same preparation logic as a real
cluster.

Steps:

1. **object from cache** — fetch the CR from the informer indexer; returns `nil` if not found (deleted between dequeue and prep)
2. **GVK fix** — watch events sometimes omit TypeMeta on the first reconcile; restored from the CRD entry
3. **namespace guard** — returns `nil` (skip) for restricted or non-allowed namespaces; deletion always passes so finalizers can be removed
4. **effective box + target** — resolve the `OperatorBox` and target from the CR's annotations
5. **normalize + base resolver** — apply `normalize.spec` templates to a deep copy and build the enrichment chain
6. **enrichment chain** — profiles, notes, intent, uniqueness, hooks external calls
7. **cross-CRD observation** — read sibling CRD CRs from peer informer caches
8. **mutation (mutateFirst)** — apply mutation rules before validation when `mutateFirst: true`
9. **validation / admission** — evaluate validation rules; a `deny` result carries the `ValidationResult` back to the caller
10. **mutation (default)** — apply mutation rules after validation (default path)

Returns `(nil, nil, nil)` when the object is not in cache or the namespace guard
blocks it. Returns `(nil, nil, err)` on failure — caller should requeue.

Kordinator is the primary caller. `simulate` also calls `Prepare` directly —
in both in-memory and envtest modes — so all simulation paths exercise the same
enrichment and guard logic as a live cluster.
This package does not import `reconciler/` — the enrichment logic is
self-contained here to avoid import cycles.
