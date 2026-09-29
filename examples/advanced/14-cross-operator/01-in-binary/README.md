# 13 — Cross-Operator Communication · 01: In Binary

Two CRDs in the same Orkestra runtime reading each other's live state — zero API server calls, zero polling loops, zero shared databases.

**What you learn:** the `cross:` block, `cross.<crd>.status.*` field paths, how Orkestra wires the Consumer's Deployment env from the Producer's live ClusterIP, and why the same-binary path costs nothing at runtime.

**Builds on:** [07 — Validation and Mutation](../../07-validation-mutation/README.md)

---

## How it works

Producer creates a Deployment and a Service. Once the Service is assigned a ClusterIP, Orkestra writes it into `status.endpoint`.

Consumer declares an `observe.cross:` block targeting `producer` by name. On every reconcile, Orkestra reads the Producer's informer cache — the same cache it already maintains — and makes the result available as `.cross.producer.*`. No HTTP call. No extra watch.

```yaml
operatorBox:
  observe:
    cross:
      - crd: producer
        selector:
          name: "{{ .metadata.name }}"
          namespace: "{{ .metadata.namespace }}"
        as: producer
```

The Consumer's Deployment then receives the endpoint as an environment variable:

```yaml
env:
  - name: PRODUCER_HOST
    value: "{{ .cross.producer.status.endpoint }}"
```

Both CRDs live in the same Katalog, the same binary, the same process.

---

## Step 1 — Validate the Katalog

```bash
ork validate
```

Expected:

```
● producer   kind: Producer / group: cross.orkestra.io / version: v1alpha1
● consumer   kind: Consumer / group: cross.orkestra.io / version: v1alpha1

2 CRDs valid
```

---

## Step 2 — Apply the CRDs

```bash
kubectl apply -f crd.yaml
```

---

## Step 3 — Install Orkestra

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --wait --timeout 120s
```

---

## Step 4 — Apply the CRs

Apply Producer first so its Service is created before Consumer reconciles:

```bash
kubectl apply -f cr-producer.yaml
kubectl apply -f cr-consumer.yaml
```

Watch the rollout:

```bash
kubectl get producers,consumers
```

```
NAME                                     IMAGE                  ENDPOINT              PHASE     AGE
producer.cross.orkestra.io/my-producer   nginx:stable-alpine    10.96.45.12:8080      Running   15s

NAME                                     IMAGE                  PRODUCER ENDPOINT     PHASE     AGE
consumer.cross.orkestra.io/my-producer   nginx:stable-alpine    10.96.45.12:8080      Running   18s
```

Consumer's `PRODUCER ENDPOINT` column matches Producer's `ENDPOINT` — Orkestra injected it without a network call.

---

## Step 5 — Verify the env injection

```bash
kubectl get deployment my-producer-deployment -o jsonpath='{.spec.template.spec.containers[0].env}' | jq .
```

```json
[
  { "name": "PRODUCER_HOST", "value": "10.96.45.12:8080" }
]
```

---

## Step 6 — Watch the cross update live

Change Producer's image to trigger a reconcile. Orkestra will re-read the cross data and re-evaluate the Consumer on its next cycle:

```bash
kubectl patch producer my-producer --type=merge -p '{"spec":{"replicas":2}}'
kubectl get consumers my-producer -o jsonpath='{.status.producerEndpoint}'
```

The endpoint stays consistent — Orkestra always reads the live informer value.

---

## Step 7 — Observe in the Control Center

```bash
ork proxy
```

Open [http://localhost:8081](http://localhost:8081). Click **Consumer** → any CR → the **Cross** panel shows `cross.producer.status.endpoint` and when it was last resolved.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
helm uninstall orkestra -n orkestra-system
```
