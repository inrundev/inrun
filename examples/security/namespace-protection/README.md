# Namespace Protection

Orkestra registers a `ValidatingWebhookConfiguration` that intercepts every `CREATE` and `UPDATE` request for CRDs managed by your Katalog. If the target namespace violates the rules declared in the Katalog, the API server rejects the request before the resource is stored — the reconciler never sees it.

Two rules are demonstrated:

| CRD | Rule | Effect |
|-----|------|--------|
| `App`, `Database` | `allowedNamespaces: [production]` | Only allowed in `production`; rejected everywhere else |
| `Cache` | `restrictedNamespaces: [kube-system, kube-public]` | Allowed anywhere except those two namespaces |

**What you learn:** `security.namespaceProtection`, `allowedNamespaces`, `restrictedNamespaces`, webhook self-healing, and observing namespace violations in the Control Center.

---

## How it works

```yaml
security:
  namespaceProtection:
    enabled: true
    cleanupOnShutdown: true
    failurePolicy: Fail
```

Per-CRD rules:

```yaml
crds:
  app:
    operatorBox:
      runtime:
        allowedNamespaces:
          - production     # whitelist — only this namespace is accepted
  cache:
    operatorBox:
      runtime:
        restrictedNamespaces:
          - kube-system    # blacklist — these namespaces are rejected
          - kube-public
```

At startup Orkestra:

1. Registers the `/namespace-protection` endpoint on its HTTPS server
2. Creates `ValidatingWebhookConfiguration` named `orkestra-namespace-protection`
3. Adds rules covering CREATE and UPDATE for all three CRD groups

`cleanupOnShutdown: true` means that on graceful shutdown, Orkestra removes the webhook configuration, its TLS Secret, and all other security resources it created — no manual cleanup required.

### One declaration. Two enforcement points.

The same rules in the Katalog are enforced at two independent points:

1. **Admission time** (`security.namespaceProtection.enabled: true`) — the webhook intercepts every `CREATE` and `UPDATE` before the object is stored. A forbidden namespace is rejected immediately; the CR never reaches etcd.

2. **Reconcile time** (always, whether webhooks are enabled or not) — `CheckNamespace` runs inside the reconciler before any child resource is created. If a CR somehow reached the API server in a forbidden namespace (e.g. webhook was absent during a restart window), Orkestra will not act on it.

> **Without `security.namespaceProtection`:** only enforcement point 2 is active. CRs in forbidden namespaces can be `kubectl apply`-ed and will appear in the API server, but Orkestra will silently skip them — no Deployments, no ConfigMaps, no status. Enabling namespace protection adds the admission gate so the bad CR is never stored in the first place.

---

## Step 1 — Install the ork CLI

```bash
curl get.orkestra.sh | bash
ork version
```

---

## Step 2 — Validate the Katalog

```bash
ork validate
```

Expected output:

```
Validating Katalog...

● app      kind: App      / group: security.orkestra.io / ...
● database kind: Database / group: security.orkestra.io / ...
● cache    kind: Cache    / group: security.orkestra.io / ...

3 CRDs valid (0 built-in, 3 custom)
```

---

## Step 3 — Apply the CRDs

If you do not have a cluster yet, run:

```bash
ork create cluster            # creates a kind cluster
```

```bash
kubectl apply -f crd.yaml
```

---

## Step 4 — Create the target namespaces

```bash
kubectl create namespace production
kubectl create namespace staging
```

---

## Step 5 — Generate and apply the operator bundle

```bash
ork generate bundle -f katalog.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

---

## Step 6 — Install Orkestra

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set gateway.enabled=true \
  --wait --timeout 120s
```

At startup you will see:

```
{"level":"info","message":"namespace protection webhook registered: orkestra-namespace-protection"}
```

---

## Step 7 — Apply allowed CRs

```bash
kubectl apply -f cr-allowed.yaml
```

This applies:
- `App/my-app` in `production` — allowed (`allowedNamespaces: [production]`)
- `Database/my-database` in `production` — allowed
- `Cache/my-cache` in `staging` — allowed (`staging` is not in `restrictedNamespaces`)

Verify:

```bash
kubectl get apps -n production
kubectl get databases -n production
kubectl get caches -n staging
```

---

## Step 8 — Try applying blocked CRs

```bash
kubectl apply -f cr-blocked.yaml
```

Two rejections, one per blocked resource:

```
Error from server: error when creating "cr-blocked.yaml": admission webhook "namespace-protect.orkestra.orkspace.io" denied the request: 

[Orkestra Security] Namespace "default" is not permitted for this CRD.

To allow this namespace, update the CRD's allowedNamespaces or restrictedNamespaces.

---
Error from server: error when creating "cr-blocked.yaml": admission webhook "namespace-protect.orkestra.orkspace.io" denied the request: 

[Orkestra Security] Namespace "kube-system" is not permitted for this CRD.

To allow this namespace, update the CRD's allowedNamespaces or restrictedNamespaces.
```

The resources are never stored. The reconciler never sees them.

### Second line of defence — the reconciler

Even if a CR bypassed the webhook (e.g., the webhook was temporarily absent during a restart window), namespace rules are enforced a second time inside the reconciler itself. `CheckNamespace` runs before any resource is created, and any CR in a forbidden namespace is silently skipped — no child resources are created, no status is written. The CR exists in the API server but Orkestra will not act on it.

This means two independent layers must both fail before a rule is violated. The webhook is the fast gate at admission time; the reconciler is the backstop if the gate is missed.

---

## Step 9 — Observe violations in the Control Center

Port-forward to the Orkestra Control Center:

```bash
ork proxy
```

Open [http://localhost:8081](http://localhost:8081).

Navigate to **namespace-protection** → click any CRD → the **Security** tab shows a log of every namespace violation: which namespace was attempted, which rule blocked it, the user identity, and the timestamp.

---

## Step 10 — Webhook self-healing

Orkestra's housekeeper watches the `ValidatingWebhookConfiguration` it owns. If it is deleted, Orkestra detects the change immediately through a Kubernetes Watch and recreates it — no restart required, no poll delay:

```bash
kubectl delete validatingwebhookconfiguration orkestra-namespace-protection
```

Watch the operator logs:

```
{"level":"warn","message":"housekeeper: configuration deleted — triggering reconcile"}
{"level":"info","message":"namespace protection webhook registered: orkestra-namespace-protection"}
```

The webhook is back within milliseconds. Namespace rules are enforced again without restarting the operator.

A safety poll (`HOUSEKEEPER_SYNC_INTERVAL`, default 30 s) continues in parallel as a backstop — it catches any drift the Watch stream might silently miss on some managed cluster distributions.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, creates namespaces, deploys the operator, applies allowed CRs, asserts that blocked CRs are rejected, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Allowed CRs accepted — deployments in production
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: my-app-app
        namespace: production
        ready: true

  - name: Blocked CR rejected by webhook
    after: cr-applied
    timeout: 30s
    commands:
      - run: kubectl apply -f ./cr-blocked.yaml
        exitCode: 1
        outputContains: "denied the request"

  - name: Deployments removed on delete
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        name: my-app-app
        namespace: production
        count: 0
```

---

## Cleanup

`cleanupOnShutdown: true` removes the webhook configuration, TLS Secret, and all other security resources automatically when Orkestra stops:

```bash
helm uninstall orkestra -n orkestra-system
```

After uninstall:
* Logs:
```json
{"level":"info","config":"orkestra-namespace-protection","time":1777692695,"message":"namespace protection webhook removed"}
{"level":"warn","time":1777692695,"message":"webhook server: offline"}
```

* Verify:
```bash
kubectl get validatingwebhookconfiguration orkestra-namespace-protection
# Error from server (NotFound): ... — removed automatically
```

```bash
chmod +x cleanup.sh && ./cleanup.sh
```