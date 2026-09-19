# Your CRD Is Enough

*The engineering argument for why the operator should have always been the CRD.*

---

In 2017, Kubernetes shipped Custom Resource Definitions as the stable mechanism for extending its API. The design was deliberate. A CRD is not a plugin — it is a declaration. You tell Kubernetes: there is a new kind of object in the world, it belongs to this group, it looks like this schema. Kubernetes responds by doing everything it already knows how to do with objects: storing them, serving them, watching them, validating them. All of the platform's machinery applies automatically to your new type.

The question CRDs left unanswered was intentional: *what happens when someone creates one of these objects?* That was the operator's job.

So the division was established. Kubernetes answers "how does this object exist?" The operator answers "what should happen when it does?" That division is correct. Kubernetes is a general-purpose platform. It should not know what a `Website` or a `Database` means.

But the ecosystem failed to ask the next question: when the API server receives a `Website` CR, what does the operator actually need to do?

Not much.

The API server already watches for changes. The schema is already declared in the CRD. The object is already stored. What the operator adds is reconciliation: watching for changes and calling the Kubernetes API to bring the actual cluster state into alignment with the declared state. That is the entire job.

Now ask what reconciliation actually requires: a watch on the resource, a queue for changes, workers to process that queue, and logic that reads the CR and creates or deletes child resources. The first three are identical for every operator ever written. They differ only in which resource is being watched and how many workers to run. The plumbing is fungible. Only the last item contains any domain-specific logic.

This is the insight Orkestra is built on. The plumbing can be provided by a runtime. The logic can be expressed as a declaration. The operator becomes a Katalog.

---

Every operator framework that came before Orkestra required the user to re-declare what the CRD had already declared. You wrote the CRD — `group: demo.orkestra.io`, `version: v1alpha1`, `kind: Website`. Then you wrote a Go struct, registered it in a scheme, generated deep-copy functions, wired the reconciler. All of this to tell the operator what the CRD had already told the cluster. The information was not new. It was already there. The framework demanded you say it again, in Go, with all the overhead that entails.

This is the root cause of operator complexity. Not the reconcile logic — that is often straightforward. The complexity is in the mandatory ceremony of re-stating what the CRD already declared.

Kubernetes stores every CRD object as `map[string]interface{}`. It does not use typed Go structs internally. The typed layer in operator frameworks was added for developer convenience and became a tax everyone paid. Orkestra removes the tax. It operates on unstructured objects because that is what the cluster stores. There is nothing to re-declare.

---

The deeper point: reconciliation is not a programming problem. It is a data problem.

The reconciler receives an object with a spec that declares desired state. It must translate that desired state into Kubernetes API calls. Every reconciler ever written does exactly this. The translation — "`spec.image` maps to `deployment.spec.containers[0].image`" — is not logic. It is a mapping. Mappings are data. Data can be declared.

```yaml
operatorBox:
  reconcile:
    onCreate:
      deployments:
        - image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
```

The template expressions are not programs. They are references. The value here comes from here. Orkestra's template resolver evaluates them against the live object and produces the literal values the registry needs. The registry — OrkestraRegistry — is the standard implementation of "given this spec, make this Kubernetes resource exist." It handles create, update, delete, owner references, idempotency, drift detection. The same code every operator would have written, extracted into a shared library.

The operator is: a mapping applied to a runtime using a library. No language required.

---

There is a common assumption that "zero code" means "lightweight." The opposite is true. In a traditional framework, an operator has whatever you built into it. If you forgot finalizers, there are no finalizers. If you forgot metrics, there are no metrics. In Orkestra, every CRD gets the full operator stack — not because you asked for it, but because the runtime provides it unconditionally. Its own informer and workqueue, its own worker pool so no other CRD can consume its capacity, its own health endpoint, five Prometheus metrics labeled by GVK, Kubernetes event emission on every operation, finalizer management, owner references on every child resource, cascade deletion, drift correction, leader election, graceful shutdown. This is more than most hand-written operators provide. Not because Orkestra is doing something magical, but because the runtime can afford to give every CRD everything when the cost of building it is shared.

Your CRD does not get a lightweight shim. It gets a super-operator.

---

CRDs were introduced so that the Kubernetes API could be extended declaratively. You declare a new resource type. Kubernetes handles the rest — storage, validation, serving, watching. The promise was never completed. Kubernetes handled the platform side. The operator side required you to leave the declarative world and write a program. The declaration ended at the CRD, and the imperative code began.

Orkestra completes the promise. You declare the CRD. You declare the Katalog. Orkestra handles the rest.

The declaration does not end at the CRD. The declaration *is* the operator.

That is what "your CRD is enough" means. Not that the CRD contains magic. But that everything Kubernetes needs to manage your resource is already present in the declaration you made. The runtime reads it. The platform enforces it. The operator emerges from the combination.

No code required.

---

If you want to see this in practice, the [Getting Started](/docs/getting-started/) guide walks through writing your first Katalog from scratch. If you want to go further, [Learning to Orkestrate](/docs/getting-started/learning-to-orkestrate/) is a progression through the example packs — from a simple declarative operator to composition, typed hooks, and multi-CRD platforms.
