# Declarative — Zero Go

The same WebApp operator as the baseline — same CRD, same Deployment, same Service, same status — expressed as a pure Katalog. No Go. No binary. No image to build.

This is the most common surprise for people coming from controller-runtime: not all operators need Go. If your operator creates Kubernetes resources, applies drift correction, writes status, and enforces admission rules, the declarative system handles all of it. The Orkestra runtime is the binary.

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/01-declarative
```

---

## What you will learn

- How to declare Deployments, Services, and status fields in a Katalog
- How `reconcile: true` enables drift correction without writing a single reconcile loop
- How admission rules work without a webhook server setup
- The complete list of what the runtime provides automatically — metrics, leader election, panic recovery, health endpoints — when you declare rather than build

---

## What changed

Everything in the controller-runtime project that was machinery is gone:

- `main.go` — not needed
- `Dockerfile` — not needed
- Helm chart — not needed
- `webapp_controller.go` — replaced by declarations in `katalog.yaml`

The reconcile logic is now a declaration:

```yaml
operatorBox:
  reconcile:
    onCreate:
      deployments:
        - name: "{{ .metadata.name }}"
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          port: "{{ .spec.port }}"
          reconcile: true
```

Template expressions — `{{ .spec.image }}`, `{{ .metadata.name }}` — have access to the full CR, its status, child resource state (`.children.*`), cross-CRD observations (`.cross.*`), HTTP call results (`.external.*`), and live runtime metrics (`.metrics.*`). See [Orkestra Notes](../../concepts/) and [Conditionals](../../concepts/conditional/) for what is expressible declaratively before reaching for Go.

---

## Try it

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/01-declarative
# Follow steps in README
```

---

## Want to add another operator?

A new business concern — a background worker, a certificate issuer, a provisioner — is just another CRD entry in the same Katalog. No new project, no new binary, no new Helm chart.

Add `crd-with-secret.yaml` and `cr-with-secret.yaml` alongside the existing files. Then add an entry under `spec.crds:`:

```yaml
spec:
  crds:
    webapp:
      # ... existing webapp entry ...

    worker:
      crdFile: ./crd-with-secret.yaml
      crFiles:
        - ./cr-with-secret.yaml
      operatorBox:
        reconcile:
          onCreate:
            secrets:
              - name: "{{ .metadata.name }}-token"
                once: true
                rotateAfter: 30d
                data:
                  token: "{{ randomAlphanumeric 32 }}"

            deployments:
              - name: "{{ .metadata.name }}"
                image: "{{ .spec.image }}"
                replicas: "{{ .spec.replicas }}"
                reconcile: true
                env:
                  - name: WORKER_TOKEN
                    valueFrom:
                      secretKeyRef:
                        name: "{{ .metadata.name }}-token"
                        key: token
        emit:
          status:
            fields:
              - path: phase
                value: "Running"
        runtime:
          allowedNamespaces:
            - default
```

This is the same pattern for all options — hooks, constructors, or mixed. The only difference is what goes inside `operatorBox:`. The CRD declaration, file layout, and the idea of adding a new entry are identical.

---

## When Go hooks become necessary

Processes that cannot be expressed as a sequence of declarative steps. Live streaming, long-running stateful workflows with runtime branching, anything the provider system does not yet cover, protocol calls that are not HTTP. When in doubt, try declaring it first — the system keeps expanding.


→ [02 — Hybrid](./03-hybrid.md)
