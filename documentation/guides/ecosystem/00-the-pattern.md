# The Pattern

The abstraction layer pattern has one principle: your teams should create resources defined in your organisation's vocabulary, not in the vocabulary of every tool you chose to run.

---

## The problem

A developer needs to deploy an application. In a platform that exposes ArgoCD directly, they write:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-webapp
  namespace: argocd
spec:
  project: default
  source:
    repoURL: "https://github.com/myorg/services"
    targetRevision: "main"
    path: "apps/webapp"
  destination:
    server: https://kubernetes.default.svc
    namespace: "production"
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

They need to know:
- That the resource goes in the `argocd` namespace, not `default`
- That `repoURL` requires the `https://` prefix
- That `project: default` is a required field
- What `syncPolicy.automated.prune` does and whether to enable it
- What `selfHeal` does and when to disable it

Now multiply this by ten tools. Each has its own spec. Each has its own namespace convention. Each has its own mandatory fields. Each has its own status format. This is what "ten mental models" means in practice.

---

## The solution

Define an internal CRD in your vocabulary:

```yaml
apiVersion: platform.myorg.io/v1alpha1
kind: App
metadata:
  name: my-webapp
spec:
  repo: "github.com/myorg/services"
  path: "apps/webapp"
  branch: "main"
  targetNamespace: "production"
  labels:
    team: platform
```

Implement the mapping once in a Katalog:

```yaml
operatorBox:
  reconcile:
    onCreate:
      custom:
        - apiVersion: argoproj.io/v1alpha1
          kind: Application
          name: "{{ .metadata.name }}"
          namespace: argocd
          spec:
            project: default
            source:
              repoURL: "https://{{ .spec.repo }}"
              targetRevision: "{{ .spec.branch }}"
              path: "{{ .spec.path }}"
            destination:
              server: https://kubernetes.default.svc
              namespace: "{{ .spec.targetNamespace }}"
            syncPolicy:
              automated:
                prune: true
                selfHeal: true
```

Now the mapping — the `https://` prefix, the `argocd` namespace, the `project: default`, the `syncPolicy` defaults — is in one place, owned by the platform team, enforced for everyone. Your developer writes `App`. Orkestra writes `Application`.

---

## What the Katalog adds beyond mapping

Mapping is the minimum. A Katalog can also:

### Admission rules

Enforce constraints on the internal CRD before it is stored:

```yaml
admission:
  validation:
    rules:
      - field: metadata.labels.team
        operator: exists
        message: "declare metadata.labels.team — all apps must declare ownership before they are deployed"
        action: deny
```

The ArgoCD Application is never created if the `App` CR does not satisfy this rule. With `security.webhooks.admission.enabled=true` .The check happens at apply time — before the reconciler runs. Without it, the check happens at reconcile time.

### Status propagation

Surface the ecosystem tool's status in the internal CRD:

```yaml
status:
  hasStatus: true   # propagate status from child resource back to CR
  fields:
    - path: syncStatus
      value: "{{ .status.sync.status }}"
    - path: health
      value: "{{ .status.health.status }}"
```

Your team runs `kubectl get apps` and sees `syncStatus: Synced` — without knowing that the data came from the ArgoCD Application's `status.sync.status` field.

### Deletion protection

Prevent accidental deletion:

```yaml
security:
  operatorBox:
    runtime:
      deletionProtection:
        enabled: true
```

The `App` CR cannot be deleted until deletion protection is explicitly disabled. This means no `kubectl delete -f` accident removes a production ArgoCD Application.

---

## The abstraction is durable

When you replace ArgoCD with Flux, the internal `App` CRD stays the same. Your developer's workflow — `kubectl apply -f app.yaml` — stays the same. The Katalog changes. The mapping changes. Nothing else changes.

This is what "the ecosystem tools keep running, they just stop being the interface" means. The platform team owns the interface. The ecosystem tools are implementation details.

The pattern is not limited to the four tools in this guide. If a tool exposes a CRD, Orkestra can manage it through an internal CRD — Strimzi topics, VictoriaMetrics clusters, Karpenter NodePools, any operator that accepts a custom resource.

---

## Status propagation

The Katalog's `status.fields` block reads live state from `.children.customs` — the informer-backed view of every custom resource the operator created. Use this to surface ecosystem resource state back onto the internal CRD without writing a controller for it.

→ [01 — ArgoCD](./01-argocd.md) — the first example in practice.
