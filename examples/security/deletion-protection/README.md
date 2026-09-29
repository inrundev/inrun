# Deletion Protection

Orkestra registers a `ValidatingWebhookConfiguration` that intercepts every `DELETE` request for CRDs managed by your Katalog and Orkestra itself. Attempts to delete a protected resource are rejected by the API server before they reach etcd. A fourth CRD — `LogStream` — is intentionally left out of the Katalog, so it can be deleted freely and demonstrates the contrast.

**What you learn:** `security.deletionProtection`, `cleanupOnShutdown`, the difference between protected and unprotected CRDs, webhook self-healing, and where to observe deletion attempts in the Control Center.

---

## How it works

```yaml
security:
  operatorBox:
    runtime:
      deletionProtection:
        enabled: true
        cleanupOnShutdown: true
        failurePolicy: Fail
```

At startup Orkestra:

1. Registers the `/deletion-protection` endpoint on its HTTPS server
2. Creates `ValidatingWebhookConfiguration` named `orkestra-deletion-protection`
3. Adds a rule that matches DELETE requests for all three CRD groups (`apps`, `databases`, `caches`)

When `failurePolicy: Fail` is set, the DELETE is also blocked if Orkestra is temporarily unreachable — you cannot accidentally delete a CR during a rolling restart.

`cleanupOnShutdown: true` means that on graceful shutdown, Orkestra removes all webhook configurations and TLS secrets it created — `orkestra-deletion-protection`, the TLS Secret, and any other security resources registered by this Katalog. This lets you decommission the operator cleanly without having to remove those resources manually.

---

## Step 1 — Install the ork CLI

```bash
curl get.orkestra.sh | bash
ork version
```

---

## Step 2 — Validate the Katalog

Always validate before applying:

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

Apply both files — the three protected CRDs and the one unprotected CRD:

```bash
kubectl apply -f crd-protected.yaml
kubectl apply -f crd-unprotected.yaml
```

---

## Step 4 — Generate and apply the operator bundle

The bundle contains the ConfigMap (your Katalog), ServiceAccount, ClusterRole, and ClusterRoleBinding:

```bash
ork generate bundle -f katalog.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

---

## Step 5 — Install Orkestra

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
{"level":"info","message":"deletion protection webhook registered: orkestra-deletion-protection"}
```

> Observe it in control center - The katalog shows "Protected"
---

## Step 6 — Apply the CRs (Optional)

```bash
kubectl apply -f cr-app.yaml
kubectl apply -f cr-database.yaml
kubectl apply -f cr-cache.yaml
kubectl apply -f cr-unprotected.yaml
```

Verify everything is running:

```bash
kubectl get apps,databases,caches,logstreams
```

```
NAME                           IMAGE        REPLICAS   AGE
app.security.orkestra.io/my-app   nginx:1.25   1          10s

NAME                                  ENGINE     STORAGE   AGE
database.security.orkestra.io/my-database   postgres   5Gi       10s

NAME                                MAXMEMORY   POLICY        AGE
cache.security.orkestra.io/my-cache   512Mi       allkeys-lru   10s

NAME                                      ENDPOINT                          FORMAT   AGE
logstream.security.orkestra.io/my-logs   https://logs.example.com/ingest   json     10s
```

---

## Step 7 — Try deleting a protected CRD

```bash
kubectl delete crd apps.security.orkestra.io
```

Expected:

```
Error from server: admission webhook "protect.crds.orkestra.orkspace.io" denied the request: 

[orkestra Security] CRD "apps.security.orkestra.io" is protected from deletion.

To delete it:
- Set security.deletionProtection.enabled: false in the Katalog
- Redeploy Orkestra Gateway, then delete the CRD.
```

The same applies to `databases.security.orkestra.io` and `caches.security.orkestra.io`. Every CRD in the Katalog is protected.

---

## Step 8 — Try deleting Orkestra's own infrastructure

Deletion protection covers Orkestra itself, not just the CRDs it manages. Every resource the Helm chart creates carries the label `orkestra.io/deletion-protection: "true"`. The webhook's second rule targets that label — any resource bearing it is protected, regardless of kind.

Try deleting the core operator resources:

```bash
# Runtime
kubectl delete deployment orkestra-runtime -n orkestra-system
kubectl delete service orkestra-runtime -n orkestra-system

# Control Center
kubectl delete deployment orkestra-cc -n orkestra-system
kubectl delete service orkestra-cc -n orkestra-system

# Namespace
kubectl delete ns orkestra-system

# Identity and permissions
kubectl delete serviceaccount orkestra -n orkestra-system
kubectl delete clusterrolebinding orkestra

# TLS secret
kubectl delete secret orkestra-internal-tls -n orkestra-system
```

Every command is denied with explanation:

```
Error from server: admission webhook "protect.resources.orkestra.orkspace.io" denied the request: 

[Orkestra Security] The Orkestra deployments "orkestra-runtime" is protected from deletion.

To disable:
- Set security.deletionProtection.enabled: false in the Katalog first.
- Redeploy Orkestra Gateway, then delete the resource.
```

If you installed Orkestra with optional components, those are protected too:

```bash
# HPA (if autoscaling is enabled)
kubectl delete hpa orkestra -n orkestra-system

# PDB (if disruption budget is enabled)
kubectl delete poddisruptionbudget orkestra -n orkestra-system

# NetworkPolicy (if network policies are enabled)
kubectl delete networkpolicy orkestra -n orkestra-system

# Ingress (if ingress is enabled)
kubectl delete ingress orkestra -n orkestra-system
```

All protected by the same webhook, same label, same policy.

---

## Step 9 — Delete the unprotected CRD

`LogStream` is not in the Katalog, so the webhook has no rule for it:

```bash
kubectl delete crd logstreams.security.orkestra.io
```

```
customresourcedefinition.apiextensions.k8s.io "logstreams.security.orkestra.io" deleted
```

Succeeds immediately.

---

## Step 10 — Observe deletion attempts in the Control Center

Every blocked deletion is recorded. Port-forward to the Orkestra Control Center to see it:

```bash
ork proxy
```

Open [http://localhost:8081](http://localhost:8081).

Click any CRD → Scroll down to **deletion-protection**. It shows a timestamped log of every blocked DELETE attempt, including the user identity and the resource that was protected.

---

## Step 11 — Webhook self-healing

Orkestra's housekeeper watches the `ValidatingWebhookConfiguration` it owns. If it is deleted, Orkestra detects the change immediately through a Kubernetes Watch and recreates it — no restart required, no poll delay:

```bash
kubectl delete validatingwebhookconfiguration orkestra-deletion-protection
```

Watch the operator logs:

```
{"level":"warn","message":"housekeeper: configuration deleted — triggering reconcile"}
{"level":"info","message":"deletion protection webhook registered: orkestra-deletion-protection"}
```

The webhook is back within milliseconds. Protection is restored automatically without restarting the operator.

A safety poll (`HOUSEKEEPER_SYNC_INTERVAL`, default 30 s) continues in parallel as a backstop — it catches any drift the Watch stream might silently miss on some managed cluster distributions.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, deploys the operator, applies CRs, asserts protection behaviour, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: App deployment created
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: my-app-app
        namespace: default
        ready: true

  - name: Protected CRD deletion blocked by webhook
    after: cr-applied
    timeout: 30s
    commands:
      - run: kubectl delete crd apps.security.orkestra.io
        exitCode: 1
        outputContains: "denied the request"

  - name: Unprotected CRD can be deleted freely
    after: cr-applied
    timeout: 30s
    commands:
      - run: kubectl delete crd logstreams.security.orkestra.io --ignore-not-found
        exitCode: 0

  - name: App deployment removed on delete
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        name: my-app-app
        namespace: default
        count: 0
```

---

## Cleanup
To be able to delete this Orkestra deployment, we need to disable deletion protection from the katalog

> [!Warn]
> `helm uninstall` also will not delete the deployment. Rather it gets stuck in `deleting` and would require manual cleanup.

### Modify the katalog
```bash
kubectl edit configmap orkestra-katalog

# or in the source katalog and reapply generated bundle
```

### Restart Orkestra
```bash
kubectl rollout restart deploy orkestra-runtime -n orkestra-system
```

Change `enabled: true` to `false` 
```yaml
security:
  operatorBox:
    runtime:
      deletionProtection:
        enabled: false
```

#### Watch Orkestra logs
```json
{"level":"info","config":"orkestra-deletion-protection","time":1777690914,"message":"deletion protection webhook removed"}
{"level":"info","namespace":"orkestra-system","time":1777690914,"message":"tls secret removed on shutdown"}
{"level":"warn","time":1777690914,"message":"webhook server: offline"}
```

```bash
kubectl get validatingwebhookconfiguration orkestra-deletion-protection

kubectl get secret orkestra-internal-tls -n orkestra-system

# Error from server (NotFound): ... — removed automatically
```

### Complete the cleanup
```bash
helm uninstall orkestra -n orkestra-system
```

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
