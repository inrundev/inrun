# 13 — Cross-Operator Communication · 02: Cross Binary

Same Producer → Consumer pattern as `01-in-binary`, but the two CRDs run in **separate Orkestra deployments** in different namespaces. Consumer reads Producer's endpoint through Orkestra's HTTP API instead of the in-process informer cache.

**What you learn:** `cross.source.endpoint`, response caching with `cacheFor`, how to run two Orkestra deployments in the same cluster, and when to use the HTTP fallback vs the zero-cost informer path.

**Builds on:** [01 — In Binary](../01-in-binary/README.md)

---

## How it works

Because Producer and Consumer are in different processes, Consumer cannot reach the Producer informer directly. The `source.endpoint` field tells Orkestra to call Producer's Orkestra REST API and cache the result:

```yaml
operatorBox:
  observe:
    cross:
      - crd: producer
        selector:
          name: "{{ .metadata.name }}"
          namespace: "{{ .metadata.namespace }}"
        as: producer
        source:
          endpoint: "http://orkestra.producer-system:8080/katalog/producer/cr/{{ .metadata.namespace }}/{{ .metadata.name }}"
          cacheFor: 15s
```

Orkestra calls that URL at most once every 15 s, regardless of resync frequency. Between calls it serves the cached value — Consumer's reconcile loop is never blocked by network latency.

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

## Step 3 — Install Orkestra in producer-system

```bash
kubectl create namespace producer-system

helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra-producer orkestra/orkestra \
  --namespace producer-system \
  --set katalog.configMapNamespace=producer-system \
  --wait --timeout 120s
```

---

## Step 4 — Install Orkestra in consumer-system

```bash
kubectl create namespace consumer-system

helm upgrade --install orkestra-consumer orkestra/orkestra \
  --namespace consumer-system \
  --set katalog.configMapNamespace=consumer-system \
  --wait --timeout 120s
```

---

## Step 5 — Apply the Producer CR

```bash
kubectl apply -f cr-producer.yaml -n producer-system
```

Wait for the endpoint to appear:

```bash
kubectl get producer my-producer -n producer-system
```

```
NAME          IMAGE                ENDPOINT           PHASE     AGE
my-producer   nginx:stable-alpine  10.96.45.12:8080   Running   12s
```

---

## Step 6 — Apply the Consumer CR

```bash
kubectl apply -f cr-consumer.yaml -n consumer-system
```

```bash
kubectl get consumer my-producer -n consumer-system
```

```
NAME          IMAGE                PRODUCER ENDPOINT   PHASE     AGE
my-producer   nginx:stable-alpine  10.96.45.12:8080    Running   8s
```

Consumer resolved the endpoint through the HTTP API. The `cacheFor: 15s` means Orkestra called `orkestra.producer-system:8080` once and will reuse the result for 15 seconds.

---

## Step 7 — Observe the API call

Check Producer Orkestra's access log or the Control Center in `producer-system`:

```bash
ork proxy -n producer-system
```

Open [http://localhost:8081](http://localhost:8081) → **Producer** → any CR → **API Calls** panel shows the Consumer cross-read requests with timestamps and cache hits.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
helm uninstall orkestra-producer -n producer-system
helm uninstall orkestra-consumer -n consumer-system
```
