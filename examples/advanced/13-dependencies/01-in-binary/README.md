# 13 — Dependencies · 01: In Binary

`App` will not start reconciling until `Database` is healthy. No init containers, no polling loops, no Go code — just a single line in the Katalog.

**What you learn:** `dependsOn`, the three `dependsOn` YAML formats, the `healthy` vs `started` conditions, how the `cross:` block reads the dependency's status for injection, and how Orkestra enforces ordering at the controller level.

<!-- **Builds on:** [13-01 — Cross Operator In Binary](../../13-cross-operator/01-in-binary/README.md) -->

---

## How it works

```yaml
app:
  dependsOn:
    database: healthy
```

Orkestra evaluates this before touching any App CR. If the Database CR with the matching name is not yet in the `healthy` condition, App's reconcile loop is skipped. There is no queue pressure, no error, no requeue storm — the reconcile simply does not start.

Once Database is healthy, App's reconcile begins and the `cross:` block reads `database.status.endpoint` to inject it as `DB_HOST`.

**Three equivalent `dependsOn` formats:**

```yaml
# Format 1 — list (condition defaults to "started")
dependsOn:
  - database

# Format 2 — key-value
dependsOn:
  database: healthy

# Format 3 — full map
dependsOn:
  database:
    condition: healthy
```

**Conditions:**
- `started` — the dependency's reconcile loop has begun at least once
- `healthy` — the dependency has reached a ready/running state (checks `status.phase: Running`)

---

## Step 1 — Validate the Katalog

```bash
ork validate
```

Expected:

```
● database   kind: Database / group: deps.orkestra.io / version: v1alpha1
● app        kind: App      / group: deps.orkestra.io / version: v1alpha1

2 CRDs valid
```

---

## Step 2 — Simulate (optional, no cluster needed)

Verify what the App reconciler would produce for this CR:

```bash
ork simulate --cr cr-app.yaml --crd app
```

```
Simulating app/my-database

  Cycle 1:
    + deployments/my-database-deployment
    + services/my-database-svc
    ~ status/my-database
  Cycle 2:
    ~ status/my-database
  (cycles 3–10: identical)

  ✓ Steady state at cycle 3 in 196ms
```

**What this means:**
- Simulate models the App reconciler in isolation — it shows what the reconciler produces when it runs. The `dependsOn` enforcement that gates the reconcile in production is a coordinator-level concern, not part of the individual reconciler.
- Cycle 1 shows the Deployment and Service the App would create once its reconcile is allowed to proceed. The `cross:` read for the database endpoint returns empty (no database in the in-memory cluster), so will be validated against a real cluster in **Step 4**.
- **Steady state at cycle 3** — the reconciler is idempotent. On a real cluster, Orkestra only runs this reconcile after the Database CR reaches `healthy`. Simulate confirms the reconciler itself is correct; the dependency gate ensures it runs at the right time.

---

## Step 3 — Apply the CRDs

```bash
kubectl apply -f crd.yaml
```

---

## Step 4 — Run Orkestra and Control Center

```bash
ork run

# Another terminal
ork contro start
```

---

## Step 5 — Apply App first (to see the wait)

Apply App before Database to observe the dependency enforcement:

```bash
kubectl apply -f cr-app.yaml
kubectl get app my-database
```

```
NAME          IMAGE                DB ENDPOINT   PHASE     AGE
my-database   nginx:stable-alpine  <none>                   5s
```

No phase written for App — Database does not exist yet. Orkestra skips its reconcile without error.

Check the control center on http://localhost:8081, you will see "Dependency Issue" for App.
Select App and scroll down to see why under `"Dependencies"`

---

## Step 6 — Apply Database

```bash
kubectl apply -f cr-database.yaml
```

Watch both:

```bash
kubectl get databases,apps
```

```
NAME                                      IMAGE               ENDPOINT                            PHASE     AGE
database.deps.orkestra.io/my-database     postgres:16-alpine  my-database.default.svc:5432        Running   10s

NAME                                 IMAGE                DB ENDPOINT                         PHASE     AGE
app.deps.orkestra.io/my-database     nginx:stable-alpine  my-database.default.svc:5432        Running   15s
```

Once Database reaches `Running`, Orkestra starts App's reconcile automatically. App picks up the endpoint from the cross block and injects it into its Deployment.

Check the control center and see App become healthy and the phase `Running`.

---

## Step 7 — Verify the injected env

```bash
kubectl get deployment my-database-deployment -o jsonpath='{.spec.template.spec.containers[0].env}' | jq .
```

```json
[
  { 
    "name": "DB_HOST",
    "value": "my-database.default.svc:5432"
  }
]
```

---

## Step 8 — Simulate a dependency restart

Delete Database CRD and watch App's behaviour:

```bash
kubectl delete crd databases.deps.orkestra.io
```
Check the control center.

Orkestra detects that the dependency is gone and puts App back into `Dependency Issue`. Re-apply Database `(crd.yaml)` and App resumes within one resync cycle.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
