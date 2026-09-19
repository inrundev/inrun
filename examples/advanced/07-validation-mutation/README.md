# 07 — Validation and Mutation

Admission-time policy without a webhook server. Defaults applied before
validation sees the object. The same rules enforced at admission and at
reconcile time from a single Katalog declaration.

**What you learn:** `validation` with `deny` and `warn`, `mutation` with
`default` and `override`, `mutateFirst`, two enforcement boundaries,
in-cluster deployment with `kubectl apply`.

**Builds on:** [06 — Basic Komposer](../../intermediate/06-komposer-basic/README.md)

---

## Two enforcement boundaries

Orkestra enforces the same declared rules at two points:

1. **Admission time** (`admission.enabled: true`) — synchronously during `kubectl apply`.

In the [katalog.yaml](katalog.yaml), admission is enabled like this:

```yaml
security:
  admission:
    webhooks:
      admission:
        enabled: true
      cleanupOnShutdown: true
```

- `admission.enabled: true` tells Orkestra to register the admission webhook.
- `cleanupOnShutdown: true` tells Orkestra to remove webhook configurations and certificates when Orkestra shuts down.

A `deny` rule rejects the request before the object is stored. The user sees the error immediately in their terminal.

2. **Reconcile time** (always) — on every reconcile cycle. Catches CRs that
existed before the rules were added, and provides enforcement even when
webhooks are disabled.

Declare once. Enforced at both points.

---

## Mutation fires before validation

At admission time, the Kubernetes API server always calls the mutating webhook
before the validating webhook. This means:

1. Mutation runs → `spec.replicas` default `"2"` is applied if absent
2. Validation runs → `spec.replicas > 0` check now sees `"2"` — passes

`mutateFirst: true` in the Katalog replicates this ordering at reconcile time.

---

## Step 1 — Deploy Orkestra in-cluster

- Apply the CRD  
  ```bash
  kubectl apply -f crd.yaml
  ```

- Generate the bundle  
  ```bash
  ork generate bundle -o bundle.yaml
  ```

- Apply the bundle  
  ```bash
  kubectl apply -f bundle.yaml
  ```

- Add the Helm repo  
  ```bash
  helm repo add orkestra https://orkspace.github.io/orkestra
  ```

- Install Orkestra  
  ```bash
  helm upgrade --install orkestra orkestra/orkestra \
    --namespace orkestra-system \
    --create-namespace \
    --set gateway.enabled=true \
    --wait --timeout 120s
  ```

> Note: Gateway is needed for webhook related operations. Deliberately decoupled from the runtime

```yaml
gateway:
  enabled: true
  endpoint: http://orkestra-gateway:8080          # The runtime advertises this endpoint for the control center to get gateway metrics
```

Webhook certificates are generated and rotated automatically by Orkestra.

> You can provide your own certificate by adding `TLS_CERT` and `TLS_KEY` to the Orkestra values.yaml file.

You should see the following endpoints as part of orkestra's startup

```text
Orkestra Endpoints:
- Health:   /health
- Ready:    /ready
- Metrics:  /metrics

Webhook Endpoints:
- Muatation:  /mutate
- Validation:  /validate
- Service Name: orkestra
- Service Namespace: orkestra-system
- Failure Policy: Ignore

- Conversion: /convert

Katalog Endpoints:
- Katalog:  /katalog
  - Website (website):  /katalog/website
  - Website (website):  /katalog/website/health
```

---

## Step 2 — Apply the valid CR

```bash
kubectl apply -f cr-valid.yaml
```

Note: `replicas` and `port` are not declared in the CR. Mutation will apply
defaults before reconcile creates resources.

```bash
# Mutation applied the defaults — spec now has replicas and port
kubectl get website my-site -o jsonpath='{.spec}' | jq
# {"environment":"production","image":"myorg/nginx:1.25","port":"8080","replicas":"2"}
```

Check status:

```bash
kubectl get website my-site -o yaml | grep -A12 "status:"
```

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
  phase: Running
  environment: production
  observedReplicas: "2"
```

---

## Step 3 — Test the deny rule

```bash
kubectl apply -f cr-invalid.yaml
```

**With webhooks enabled:**
```
Error from server: admission webhook "validate.orkestra.orkspace.io" denied the request:
[orkestra] validation failed: field "spec.image": images must come from the internal registry (myorg/)
```

The CR is never stored. Nothing to clean up.

**Without webhooks:**

The CR is stored, but reconcile halts and two conditions are written:

```bash
kubectl get website bad-site -o yaml | grep -A6 "conditions:"
# - type: Ready             status: "False"  reason: ReconcileError
# - type: ValidationFailed  status: "True"   reason: DenyRuleViolation
#   message: "validation denied: field \"spec.image\": images must come from the internal registry (myorg/)"
```

`ValidationFailed=True` is the signal to watch — it is set only when a `deny` rule fires, making it safe to gate CI or alerting on this condition type specifically. Both conditions are also visible in the Control Center **Conditions** tab without reading logs.

---

## Step 4 — Test the warn rule

Apply a CR without `spec.environment`:

```bash
kubectl apply -f - <<EOF
apiVersion: demo.orkestra.io/v1alpha1
kind: Website
metadata:
  name: warn-site
  namespace: default
spec:
  image: myorg/nginx:1.25
  replicas: 1
  port: "8080"
  # environment not declared → warn rule fires
EOF
```

**With webhooks:** `kubectl apply` shows a warning but proceeds:
```
Warning: orkestra: field "spec.environment": declare spec.environment for better observability
website.demo.orkestra.io/warn-site created
```

**Without webhooks:** reconcile proceeds but `ValidationWarning=True` is written to the CR status and visible in the Control Center **Conditions** tab:

```bash
kubectl get website warn-site -o yaml | grep -A5 "ValidationWarning"
# - type: ValidationWarning  status: "True"  reason: WarnRuleViolation
#   message: "field \"spec.environment\": declare spec.environment for better observability"
```

---

## Step 5 — Check the metrics

```bash
ork proxy

curl localhost:8080/katalog/website | jq '{
  "reconcileTotal": .reconcileTotal,
  "admission": .admission
}'
```

---

## Cleanup

`cleanupOnShutdown: true` removes both webhook configurations, the TLS Secret, and all other security resources automatically when Orkestra stops:

```bash
helm uninstall orkestra -n orkestra-system
```

After uninstall:
* Logs:
```json
{"level":"info","config":"orkestra-admission-validation","time":1777697002,"message":"webhook: ValidatingWebhookConfiguration unregistered"}
{"level":"info","config":"orkestra-admission-mutation","time":1777697002,"message":"webhook: MutatingWebhookConfiguration unregistered"}
{"level":"info","namespace":"orkestra-system","time":1777697002,"message":"tls secret removed on shutdown"}
{"level":"warn","time":1777697002,"message":"webhook server: offline"}
```

* Verify:
```bash
kubectl get validatingwebhookconfiguration orkestra-admission-validation
# Error from server (NotFound): ... — removed automatically

kubectl get mutatingwebhookconfiguration orkestra-admission-mutation
# Error from server (NotFound): ... — removed automatically

kubectl get secret orkestra-tls -n orkestra-system
# Error from server (NotFound): ... — removed automatically
```

### Cleanup CR
```bash
chmod +x cleanup.sh && ./cleanup.sh
```