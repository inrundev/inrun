# 03 — Cross-CRD Dependency (cross:)

Two CRDs. `ManagedDatabase` creates a Postgres Deployment and writes its endpoint to status. `DatabaseBackedApp` observes the `ManagedDatabase` CR via `cross:` — it reads from the informer cache, zero API calls — and waits for it to reach `Ready` before creating its own Deployment. The database endpoint flows automatically into a ConfigMap.

**What you learn:** `cross:` observation, `dependsOn:` worker ordering, reading from `.cross.<name>.status.*` in templates, and how two CRDs coordinate without any direct communication.

---

## Step 1 — Validate

[`katalog.yaml`](katalog.yaml) has the `managed-database` CRD block commented out by default. This lets the root `full-stack-app/` composite import it without a duplicate-CRD conflict. When running this example in isolation, uncomment the `managed-database:` block first:

```yaml
# katalog.yaml — uncomment this before running ork validate from 03-cross-crd/
spec:
  crds:
    managed-database:
      crdFile: ./crd-managed-database.yaml
      ...
```

Then validate:

```bash
ork validate
```

Expected:
```
✓ managed-database
    kind: ManagedDatabase
    group: advanced.orkestra.io / version: v1alpha1 / plural: manageddatabases
✓ database-backed-app
    kind: DatabaseBackedApp
    group: advanced.orkestra.io / version: v1alpha1 / plural: databasebackedapps
    dependsOn: managed-database
```

---

## Step 2 — Start the runtime

`ork run` reads the `crdFile` entries declared in `katalog.yaml`, applies both CRDs to the cluster, and starts the runtime:

```bash
ork run
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **cross-crd** to see both CRDs.

---

## Step 4 — Apply the database CR first

```bash
kubectl apply -f database-cr.yaml
```

Wait for the ManagedDatabase to reach `Ready`:

```bash
kubectl get manageddatabase my-app-db
# NAME         PHASE   ENDPOINT
# my-app-db    Ready   my-app-db-postgres.default.svc.cluster.local:5432
```

The `ManagedDatabase` reconciler created a Postgres Deployment and Service, then wrote the endpoint to status.

---

## Step 5 — Apply the application CR

```bash
kubectl apply -f application-cr.yaml
```

Wait one reconcile. The `DatabaseBackedApp` sees the `ManagedDatabase` CR is `Ready` and creates its Deployment and ConfigMap.

```bash
kubectl get databasebackedapp my-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  databaseEndpoint: my-app-db-postgres.default.svc.cluster.local:5432
```

```bash
kubectl get deploy my-app
kubectl get configmap my-app-config -o yaml
```

The ConfigMap contains `DB_HOST` populated from the database CR's status — written by Orkestra on every reconcile, always current.

---

## Step 6 — The informer cache

The `cross:` observation reads from the in-memory informer cache — no `client.Get()`, no API call. Delete and re-apply `database-cr.yaml` and watch `DatabaseBackedApp` transition to `WaitingForDatabase`, then back to `Ready` as the database recovers.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies both CRDs, sets up the database dependency, starts the operator, applies the application CR, asserts the cross: read injects the endpoint, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Deployment ready after database is available
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: my-app
        namespace: default
        ready: true

  - name: Status reflects database endpoint
    after: cr-applied
    timeout: 90s
    commands:
      - run: kubectl get databasebackedapp my-app -o jsonpath='{.status.databaseEndpoint}'
        outputContains: my-app-db
```

---

## Cleanup

```bash
kubectl delete databasebackedapp my-app --ignore-not-found
kubectl delete manageddatabase my-app-db --ignore-not-found
kubectl delete -f crd-managed-database.yaml -f crd-database-backed-app.yaml --ignore-not-found
```
