# Getting Started

Orkestra turns a YAML file into a working Kubernetes operator. This guide takes you from zero to a running operator — no Go, no code generation.

---

!!! tip "New to Kubernetes?"
    Read [Kubernetes, operators, and Orkestra](./00-kubernetes-basics.md) first — CRDs, the reconcile loop, and where Orkestra fits in.

## Requirements

- A running Kubernetes cluster — [kind](https://kind.sigs.k8s.io/) recommended for local development
- [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) configured and pointing at your cluster

No cluster yet? Use `ork run --dev` below — it creates a local kind cluster automatically.

---

## Install

**macOS (Homebrew)**

```bash
brew install orkspace/tap/ork orkspace/tap/orkcc
```

This installs two binaries: `ork` (the operator CLI) and `orkcc` (the Control Center).

**Linux / macOS (curl)**

```bash
curl -sSL https://get.orkestra.sh | bash
```

**Verify**

```bash
ork version
```

---

## Your First Operator

**Step 1 — Scaffold a project**

```bash
ork init
```

`ork init` creates a `katalog.yaml` in the current directory, ready to run.

**Step 2 — Start the runtime**

If you have a running cluster:

```bash
ork run
```

If you have no cluster yet, add `--dev` — Orkestra creates a kind cluster automatically:

```bash
ork run --dev
```

You will see:

```text
INFO  CRD applied                crd=websites.demo.orkestra.io
INFO  CR applied                 name=hello-website namespace=default
INFO  Informer synced            crd=website
INFO  Workers started            crd=website  workers=3
INFO  Health server ready        addr=:8080
```


**Step 3 — Reconciliation**

Watch Orkestra's output in the first terminal. You'll see the reconcile event arrive and the Deployment get created.

**Step 4 — Verify**

```bash
kubectl get deployments
kubectl get pods
```

A Deployment named `hello-website-deployment` appears. Orkestra set owner references — deleting the `Website` CR will cascade-delete it.

**Step 5 — Open the Control Center**

In a second terminal:

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) to see the live operator — CRD health, worker state, reconcile metrics, queue depth.

---

## What the Katalog Looks Like

The katalog at `katalog.yaml`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: hello-website

spec:
  crds:
    website:
      crdFile: ./crd.yaml
      crFiles:
        - ./cr.yaml

      operatorBox:
        reconcile:
          onCreate:
            deployments:
              - image: "{{ .spec.image }}"
```

`crdFile` points to the CRD definition. Orkestra reads it, infers the API types, and applies it to the cluster automatically when `ork run` starts. No separate `kubectl apply -f crd.yaml` step.

When the CR is applied, Orkestra resolves `{{ .spec.image }}` from the live CR and creates the Deployment.

---

## Built-in Health Endpoints

While the operator is running:

```bash
# Health check for the website CRD
curl localhost:8080/katalog/website/health | jq

# Full detail — workers, queue, reconcile stats
curl localhost:8080/katalog/website | jq

# All managed CRDs
curl localhost:8080/katalog | jq

# Prometheus metrics
curl localhost:8080/metrics
```

---

## Cleanup

```bash
# Delete the CR — cascades to the Deployment
kubectl delete -f cr.yaml

# Stop the operator
Ctrl+C
```

---

## CLI Reference

| Command | Description |
|---------|-------------|
| `ork init` | Scaffold a new operator project in the current directory |
| `ork run -f <path>` | Start the runtime. `<path>` is optional if `katalog.yaml` or `komposer.yaml` is in the current directory |
| `ork run --dev` | Start the runtime, create kind cluster if needed |
| `ork validate` | Validate a Katalog without starting |
| `ork template` | Preview the merged, resolved Katalog |
| `ork control` | Start the Control Center at localhost:8081 |
| `ork version` | Print version |

---

## Next Steps

- **[Writing Your First Motif](./02-writing-your-first-motif.md)** — reusable building blocks for your operators
- **[Writing Your First Katalog](./03-writing-your-first-katalog.md)** — build your own operator from scratch
- **[Writing Your First Komposer](./04-writing-your-first-komposer.md)** — compose multiple Katalogs
- **[Basic Reconciliation](./05-basic-reconciliation.md)** — understand the full reconcile lifecycle
- **[Writing Your First E2E Test](./06-writing-your-first-e2e.md)** — declarative end-to-end testing
- **[Running Patterns from the Registry](./07-running-patterns.md)** — inspect, pull, and run published patterns with a single command
- **[Learning to Orkestrate](./01-learning-to-orkestrate/index.md)** — progression through the example packs
- **[Kubernetes, operators, and Orkestra](./00-kubernetes-basics.md)** — CRDs, reconciliation, and where Orkestra fits
