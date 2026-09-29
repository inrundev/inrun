# 13 — Dependencies · 02: Cross Binary

Same `dependsOn: database: healthy` ordering as `01-in-binary`, but Database runs in a **separate Orkestra deployment** in a hardened namespace. App's Orkestra resolves the dependency condition through the Database Orkestra's HTTP health API.

**What you learn:** Cross-deployment dependency resolution, namespace isolation for sensitive workloads (e.g. a database operator with restricted RBAC), and how `source.endpoint` carries dependency checks across process boundaries.

**Builds on:** [01 — In Binary](../01-in-binary/README.md)

---

## How it works

App's `observe.cross:` block includes `source.endpoint` pointing at the Database Orkestra:

```yaml
operatorBox:
  observe:
    cross:
      - crd: database
        selector:
          name: "{{ .metadata.name }}"
          namespace: "{{ .metadata.namespace }}"
        as: database
        source:
          endpoint: "http://orkestra.db-system:8080/katalog/database/cr/{{ .metadata.namespace }}/{{ .metadata.name }}"
          cacheFor: 15s
```

When `dependsOn: database: healthy` is evaluated, Orkestra checks `cross.database.status.phase`. If the remote call returns `phase: Running`, the condition is satisfied and App's reconcile begins.

---

## Step 1 — Validate the Katalog

```bash
ork validate
```

---

## Step 2 — Apply the CRDs (cluster-wide, once)

```bash
kubectl apply -f crd.yaml
```

---

## Step 3 — Install Orkestra in db-system

```bash
kubectl create namespace db-system

helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra-db orkestra/orkestra \
  --namespace db-system \
  --set katalog.configMapNamespace=db-system \
  --wait --timeout 120s
```

---

## Step 4 — Install Orkestra in app-system

```bash
kubectl create namespace app-system

helm upgrade --install orkestra-app orkestra/orkestra \
  --namespace app-system \
  --set katalog.configMapNamespace=app-system \
  --wait --timeout 120s
```

---

## Step 5 — Apply Database CR

```bash
kubectl apply -f cr-database.yaml -n db-system
kubectl get database my-database -n db-system
```

```
NAME          IMAGE               ENDPOINT                       PHASE
my-database   postgres:16-alpine  my-database.db-system.svc:5432  Running
```

---

## Step 6 — Apply App CR

```bash
kubectl apply -f cr-app.yaml -n app-system
kubectl get app my-database -n app-system
```

```
NAME          IMAGE                DB ENDPOINT                      PHASE
my-database   nginx:stable-alpine  my-database.db-system.svc:5432   Running
```

App resolved the dependency through the HTTP API, then injected the endpoint. The two Orkestra deployments never share a process or a kubeconfig context.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
helm uninstall orkestra-db  -n db-system
helm uninstall orkestra-app -n app-system
```
