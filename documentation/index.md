# Why Orkestra

Building a Kubernetes operator means writing Go: controllers, informers, reconcilers, finalizers, events, metrics. Every team does this. Every team builds the same infrastructure. The only part that differs is the business logic — and it is usually just 10 lines of YAML describing what you want.

Orkestra inverts this. You write a **Katalog** — a declarative YAML file that describes your CRDs and what should happen when a Custom Resource is created, updated, or deleted. Orkestra provides the runtime: informers, worker pools, drift correction, finalizers, events, metrics, health endpoints.

**No Go. No code generation. No controller boilerplate.**

---

## What You Write

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: webapp-operator
spec:
  crds:
    webapp:
      crdFile: ./crd.yaml
      operatorBox:
        reconcile:
          onCreate:
            deployments:
              - image: "{{ .spec.image }}"
                replicas: "{{ .spec.replicas }}"
                reconcile: true
            services:
              - port: "80"
                targetPort: "{{ .spec.port }}"
                reconcile: true
```

That is a fully working operator. Orkestra reads the CRD from `crdFile`, applies it to the cluster, starts informers, and reconciles every `WebApp` CR into a Deployment and Service. Drift correction included.

---

## What Orkestra Does

When you run `ork run`:

1. Reads your Katalog and finds `crdFile: ./crd.yaml`
2. Applies the CRD to the cluster automatically
3. Starts informers watching for CRs of that type
4. For each CR: resolves templates, creates resources, adds finalizers
5. On every reconcile: corrects drift on resources marked `reconcile: true`
6. On CR delete: cleans up all child resources

All of this from a YAML file.

---

## Who It Is For

- **Platform engineers** building internal developer platforms without writing operators
- **SREs** who want declarative infrastructure, not Go code
- **Application teams** defining their own CRDs without learning controller patterns

---

## Get Started

```bash
curl -sSL https://get.orkestra.sh | bash
ork init
ork run
# Orkestra reads katalog.yaml from the current directory and starts the runtime.
```

→ [Getting Started](./getting-started/index.md)

---

## Further Reading

- [Your CRD Is Enough](/blog/your-crd-is-enough/) — the idea behind Orkestra
- [Why Orkestra](/blog/why-orkestra/) — a deeper look at the problem
- [FAQs](./faqs/index.md) — common questions, including comparisons with Kubebuilder and Helm
