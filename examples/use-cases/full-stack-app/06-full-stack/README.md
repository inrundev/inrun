# 06 — Full Stack (all patterns combined)

One CR, all five patterns at once. A `FullStackApp` creates 3 regional Deployments (`forEach`), a generated Secret (`once:`), a ConfigMap sourced from a database CR (`cross:`), all gated on a health check (`external:`), with a cleanup Job on terminal phases (`or:`). This is the showcase — everything Orkestra can do in a single declaration.

---

## Step 1 — Validate

[`katalog.yaml`](katalog.yaml) has the `managed-database` CRD block commented out by default. This lets the root `full-stack-app/` composite import it without a duplicate-CRD conflict. When running this example in isolation, uncomment the `managed-database:` block first:

```yaml
# katalog.yaml — uncomment this before running ork validate from 06-full-stack/
spec:
  crds:
    managed-database:
      crdFile: ../03-cross-crd/crd-managed-database.yaml
      ...
```

Then validate:

```bash
ork validate
```

Expected:
```
● full-stack-app
    kind: FullStackApp / group: advanced.orkestra.io / version: v1alpha1
    mode: dynamic / workers: 2 / resync: 30s

● managed-database
    kind: ManagedDatabase / group: advanced.orkestra.io / version: v1alpha1
    mode: dynamic / workers: 2 / resync: 30s

────────────────────────────────────────────────────────────
2 CRDs valid (0 built-in, 2 custom)
```

> If you plan to run the full composite from the parent directory (`full-stack-app/`), comment `managed-database:` back out before switching — see [Troubleshooting](../README.md#troubleshooting).

---

## Step 2 — Start the runtime

`ork run` reads the `crdFile` declared in `katalog.yaml`, applies the other dependencies defined in the `setup` block — the managed-database CRD and its seed CR — and starts the runtime. `--dev-server` is required — the health check calls `/health` on every reconcile:

```bash
ork run --dev-server
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **full-stack-app**, then select the **FullStackApp** CRD.

---

## Step 4 — Apply the CR

`cr.yaml` includes both the `ManagedDatabase` dependency and the `FullStackApp` — one apply is enough:

```bash
kubectl apply -f cr.yaml
```

The FullStackApp spec covers all five patterns:

```yaml
spec:
  image: nginx:1.25
  regions: [us-east-1, eu-west-1, ap-southeast-1]   # forEach
  serviceUrl: http://localhost:9999                   # external:
  environment: production                             # ConfigMap value
  notify: "true"                                     # or: notify trigger
```

---

## Step 5 — Watch the lifecycle

Phase walks forward as each condition is satisfied:

```bash
kubectl get fullstackapp my-app -w
# NAME     PHASE
# my-app   Pending
# my-app   WaitingForDatabase      ← cross: observed, database not yet Ready
# my-app   Ready                   ← health check passed, database Ready
```

Check everything that was created:

```bash
kubectl get deploy,secret,configmap -l orkestra-owner=my-app
```

Expected — for one CR:
```
deployment.apps/my-app-us-east-1      1/1
deployment.apps/my-app-eu-west-1      1/1
deployment.apps/my-app-ap-southeast-1 1/1
secret/my-app-creds                   (generated once, rotates after 90d)
configmap/my-app-config               (DB_HOST from database CR status)
```

---

## Step 6 — Status tells the full story

```bash
kubectl get fullstackapp my-app -o yaml | grep -A16 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  regionsDeployed: "3"
  databaseEndpoint: my-app-db-postgres.default.svc.cluster.local:5432
  credentialsSecret: my-app-creds
```

Five patterns. One CR. Zero Go.

---

## Cleanup

```bash
kubectl delete fullstackapp my-app --ignore-not-found
kubectl delete manageddatabase my-app-db --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
```
