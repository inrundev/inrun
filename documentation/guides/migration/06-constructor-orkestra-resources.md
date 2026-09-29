# Constructor — Orkestra Resources

Option 04 lifted the reconcile logic unchanged and removed the machinery. Option 05 goes one step further: the Get → IsNotFound → Create → Patch pattern inside the reconciler is replaced by `pkg/resources`. The constructor signature stays the same — only the resource management methods change.

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/06-constructor-orkestra-resources
```

---

## What you will learn

- How `orkdeploy.Update`, `orksvc.Update`, `orksa.Update` replace the manual resource management pattern
- What `DeleteIfOwned` does and when to use it
- The difference between this and the hooks-only pattern — same `pkg/resources` library, different call site

---

## What changed

The constructor, struct, and signature are identical to option 04. The sub-methods that previously did Get / IsNotFound / Create / Patch are replaced by a single `Update` call per resource.

`Update` handles: create if absent, patch if drifted, owner references, system labels. Same result, less code, no logic you need to maintain.

---

## When to pick this over option 04

After option 04 is working and you want to clean it up. Option 04 is the safer first step — it minimises the diff so you can verify the operator behaves correctly under Orkestra before changing more. Option 05 is the cleaner long-term form.

---

## Try it

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/05-constructor-orkestra-resources
# Follow steps in README
```

→ [ork migrate](./07-ork-migrate.md) — automate the option 04 path for your own operator.
