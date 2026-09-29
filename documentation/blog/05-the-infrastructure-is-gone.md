# The Infrastructure is Gone

*Orkestra does not give you a better way to build operator infrastructure. It removes it.*

---

Every operator author starts in the same place. Before a single line of business logic, there is a list of things to build and maintain that have nothing to do with the problem you actually came to solve.

You need a reconciler. A controller loop. Leader election. A webhook server for admission and conversion. A build pipeline that produces a container image. A Helm chart to deploy it. A registry to store the image. A way to version the operator and upgrade it in production without downtime. A way to test it against a real cluster before it ships. A way to deprecate old versions without breaking consumers. A way to protect your CRs from accidental deletion.

None of that is your business logic. All of it must exist before your business logic can run.

This is the operator infrastructure problem. It is not unique to any one framework. It is the tax every operator author pays.

---

## The separation Orkestra makes

Orkestra's answer is not a better way to build operator infrastructure. It is a separation: infrastructure on one side, declaration on the other.

Orkestra handles the infrastructure. You handle the declaration.

The infrastructure side is the runtime, the gateway, the control center, the CLI — and critically, the registry. The registry is where patterns live between production and consumption. It is where `ork push` deposits proof of what was tested and how. It is where `ork pull` retrieves patterns with that proof intact. The registry is not an afterthought; it is the distribution model. The lifecycle follows from it.

None of this runs in your cluster as a new stack of CRDs. The separation is clean. Orkestra takes the infrastructure. You never touch it.

---

## What remains

Once the infrastructure is gone, what is left is a declaration.

Orkestra gives you five ways to make it:

**No code.** Write a Katalog. Declare your CRD, the resources to create on `onCreate`, the status fields to write, the validation rules to enforce, the dependencies to respect. The reconciler is not yours to write — it is Orkestra's, and it runs your declaration. No Go, no build pipeline, no image.

**Hybrid.** Combine declarative and hooks in a single operator. Declare what templates can express; attach hooks for what they cannot. The boundary is yours to draw.

**Hooks.** Attach Go functions at specific points in the reconcile cycle. The declarative model handles everything it can; your hooks handle what it cannot. You write exactly the code your business logic requires, nothing else.

**Constructor.** Replace the reconciler entirely with your own. Full control, typed input, Orkestra's runtime as the host. The infrastructure is still gone — you are only responsible for the domain logic.

**Constructor with Orkestra resources.** Write your own reconciler and use Orkestra's resource helpers — `orkdeploy.Update`, `orksvc.Update`, and the rest of `pkg/resources` — instead of raw client Get → IsNotFound → Create → Patch sequences. Owner references and drift correction are handled automatically. The constructor signature stays yours; the resource management layer becomes Orkestra's.

---

## One problem

Whichever path you take, the shape of what you are solving is the same: your business logic. Not the watch. Not the queue. Not the leader election, the build pipeline, the webhook server, or the upgrade path.

Just the problem you came to solve.

That is what Kubernetes intended when it introduced CRDs. The CRD was the declaration. The operator was supposed to answer one question about it: what should happen when someone creates one of these? We turned that one question into years of infrastructure work.

Orkestra gives the question back. Whichever you pick — no code, hybrid, hooks, constructor, constructor with Orkestra resources, or a remote HTTP server in any language — you are solving exactly one problem.

Yours.

---

**Want to try it?**

```bash
ork init --pack from-controller-runtime
```
