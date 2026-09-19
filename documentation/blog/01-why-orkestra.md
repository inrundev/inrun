# Why Orkestra?

Kubernetes has always promised declarative infrastructure. You describe what you want. The platform makes it so.

That promise holds everywhere — until you need to extend Kubernetes itself.

The moment you need a custom resource, you leave the declarative world. You write Go. You scaffold controllers. You wire informers and schemes. You manage reconcile loops, retries, and backoff. You build and push images. You maintain a project whose entire purpose is to watch another project.

Every major operator framework to date has accepted this as the cost of entry. Kubebuilder, Operator SDK, Metacontroller — they each make the Go easier. None of them make it unnecessary.

This creates a paradox: **to make Kubernetes more declarative, you must write imperative code.**

Orkestra breaks that paradox.

---

Understanding why Orkestra exists requires tracing honestly how the operator pattern evolved — not to criticise what came before, but to see how each improvement brought its own new problem.

The original operator pattern was elegant in concept: a controller that watches a custom resource and continuously reconciles the cluster toward the desired state. The implementation required deep familiarity with client-go internals — informers, workqueues, schemes, REST mappers. Most of that implementation was identical across operators. The business logic was a small fraction of the total code.

Kubebuilder and Operator SDK solved the boilerplate problem. Code generation, scaffolding, and controller-runtime reduced hundreds of lines to dozens. The cost of entry dropped meaningfully. But it did not reach zero. The generated project still required Go, a build pipeline, an image registry, and a deployment manifest. Adding a new CRD meant a new type, new code generation, a new binary, a new deployment.

The more pressing problem was what this model produced at scale. A typical production cluster runs operators for Prometheus, Cert Manager, Ingress, External Secrets, and a handful of internal CRDs — each a separate binary, a separate deployment, a separate failure domain, a separate upgrade cadence. The operator became a software project. The CRD became an afterthought.

The fundamental problem was never the boilerplate, or the language, or the build pipeline. **The fundamental problem was that every CRD demanded its own operator, and every operator was invented from scratch.**

Orkestra is the shared infrastructure that changes this.

---

The Kubernetes community has long held that the right design is one operator per CRD. Orkestra agrees completely. The reframe is not about sharing a reconciler across CRDs — it is about what "one operator" actually means.

In traditional frameworks, one operator means one Go binary, one deployment, one informer, one workqueue. When you have twelve CRDs, you have twelve of everything. The per-CRD isolation exists, but so does twelve times the operational overhead.

In Orkestra, each CRD is still its own operator — fully isolated, fully independent. It has its own informer, its own worker pool, its own workqueue, its own health endpoint, its own metrics, its own reconciler, its own failure domain. A panic in one CRD's reconciler does not affect any other. What is shared is not the operator logic. What is shared is the **orchestration infrastructure** — the runtime that hosts these operators, starts them in dependency order, manages their lifecycle, and provides the observability layer that makes them all look consistent.

Your CRD does not get a lightweight reconciler. It gets a full operator — with every capability that would have taken weeks to write — for free, as a consequence of being declared in a Katalog. This is what we mean by the super-operator model: each CRD becomes a complete, production-grade operator. They just share the infrastructure that makes that possible.

---

In practice, you write a Katalog:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
spec:
  crds:
    website:
      crdFile: website.yaml
      operatorBox:
        reconcile:
          workers: 3
      crFiles:
        - examples/website-sample.yaml
      setup:
        - rbac/namespace.yaml
```

`crdFile` is your CRD definition. `crFiles` are sample CRs to apply. `setup` is a list of manifests Orkestra applies to the cluster before the operator starts — namespaces, roles, whatever the operator needs to exist first. You run `ork run`. Orkestra applies all three and starts the runtime.

The `Website` CRD now has its own informer, dedicated workers, independent workqueue, a full `/katalog` API with a health endpoint at `/katalog/website/health`, five Prometheus metrics labeled by GVK, Kubernetes events on every operation, finalizer management, owner references, cascade deletion, drift correction, and leader election — out of the box. No Go. No build pipeline.

The super-operator model holds even for advanced cases. Multi-version CRDs — which traditionally require writing conversion functions in Go, deploying a webhook server, and managing TLS certificates — are handled declaratively in Orkestra. Each version gets its own complete operator stack. Conversion paths between versions are declared in the Katalog alongside the reconcile templates. No Go. No webhook binary. No certificates to manage.

When your operator is a Katalog, it inherits all the properties of data. It can be versioned, diffed, and composed. A Komposer can assemble a full platform control plane from Katalogs pulled from the registry, local files, or Helm charts — with inline overrides for environment-specific configuration. Platform teams publish. Application teams compose. Environments are different configurations of the same operator definitions. The same model Helm brought to deployment manifests, applied to the operators themselves.

---

The operator pattern is the right abstraction for Kubernetes extensibility. The requirement to implement it in Go, built and deployed separately for each CRD, has been a constraint of convention rather than necessity.

When the same runtime can host any number of CRDs — each isolated, each complete, each observable through a consistent interface — the operator becomes something you declare, not something you build.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The same principle, applied one level up.
It was always possible. It just needed someone to build it.

---

If you want to see this in practice, the [Getting Started](/docs/getting-started/) guide walks through writing your first Katalog from scratch. If you want to go further, [Learning to Orkestrate](/docs/getting-started/learning-to-orkestrate/) is a progression through the example packs — from a simple declarative operator to composition, typed hooks, and multi-CRD platforms.
