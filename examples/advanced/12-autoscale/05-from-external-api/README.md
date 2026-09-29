# 05 — Autoscale: From External API

This example builds on **[04-sibling-in-cluster](../04-sibling-in-cluster/README.md)**.

In 04, the autoscaler read metrics from a sibling Orkestra runtime via an HTTP endpoint. Here, the source is not an Orkestra runtime at all — it is an external workload exposing a metrics endpoint.

The scenario: a payment processor operator that scales based on the payment system's queue depth. The payment system runs on its own endpoint. The processor operator has no direct connection to it other than the HTTP call Orkestra makes on each autoscale evaluation.

---

## What You Learn

- How to autoscale based on any external HTTP endpoint
- How `source.endpoint` in autoscale conditions works for non-Orkestra sources
- The expected response shape: `{ "metrics": { ... } }`
- How to simulate load with the Orkestra dev server

---

## Prerequisites

- Ork CLI
- Kubernetes cluster (Kind recommended)

Install Ork CLI:

```bash
curl get.orkestra.sh | bash
```

---

## Run the Example

### 1. Start Orkestra with the dev server

```bash
ork run --dev-server
```

The dev server starts on port `9999` and serves the payment system metrics at `/autoscale-metrics`.

### 2. Apply CRD and CR

```bash
kubectl apply -f crd.yaml
kubectl apply -f cr.yaml
```

### 3. Start the Control Center

```bash
ork control
# username:password → orkestra
# → http://localhost:8081
```

You will see the Processor operator running at baseline — 2 workers.

### 4. Check the current metrics

```bash
curl -s http://localhost:9999/autoscale-metrics | jq .
```

```json
{
  "metrics": {
    "errorRatePercent": 0,
    "queueDepth": 12,
    "reconcileDurationP95Ms": 142.3,
    "workersBusyPercent": 18,
    "workersIdlePercent": 82
  }
}
```

Queue depth is 12 — well below the threshold of 60. Processor stays at baseline.

### 5. Flip the load

```bash
curl -X POST http://localhost:9999/autoscale-metrics/flip
```

The payment system is now overloaded:

```json
{
  "metrics": {
    "errorRatePercent": 0,
    "queueDepth": 98,
    "reconcileDurationP95Ms": 1244.18,
    "workersBusyPercent": 100,
    "workersIdlePercent": 0
  }
}
```

Queue depth is 98 — above the threshold. On the next autoscale evaluation (20s interval), Processor scales up:

```json
{"level":"info","crd":"external.orkestra.io/v1alpha1, Kind=Processor","workers":8,"message":"autoscaler: worker pool resized"}
{"level":"info","crd":"external.orkestra.io/v1alpha1, Kind=Processor","queueDepth":800,"message":"autoscaler: queue depth limit updated"}
```

Control Center shows Processor scaled: workers 2 → 8. Confirm from the Orkestra katalog API:

```bash
curl -s http://localhost:8080/katalog/processor | jq .metrics.workers
# 8
```

### 6. Flip back

```bash
curl -X POST http://localhost:9999/autoscale-metrics/flip
```

Payment system queue drops back to 12. After the **3 minute cooldown**, Processor returns to baseline:

```json
{"level":"info","crd":"external.orkestra.io/v1alpha1, Kind=Processor","workers":2,"message":"autoscaler: worker pool resized"}
{"level":"info","crd":"Processor","workers":2,"queueDepth":100,"message":"autoscaler: baseline restored"}
```

---

## How It Works

The katalog declares the external source in the `observe.cross:` block, and the autoscale condition references it by field path — no repeated endpoint:

```yaml
operatorBox:
  observe:
    cross:
      - crd: payment-system
        source:
          endpoint: "http://localhost:9999/autoscale-metrics"
          cacheFor: 10s
        as: paymentSystem

autoscale:
  interval: 20s
  cooldown: 3m
  conditions:
    when:
      - field: cross.paymentSystem.metrics.queueDepth
        greaterThan: "60"
  do:
    workers: 8
    queueDepth: 800
    resync: 10s
```

`cross.paymentSystem.metrics.queueDepth` — Orkestra finds the `payment-system` cross entry, calls its endpoint, and reads `queueDepth` from the `metrics` object in the response. The endpoint can be any workload that exposes this shape.

The response shape Orkestra expects:

```json
{
  "metrics": {
    "queueDepth": 98
  }
}
```

`cacheFor: 10s` — the response is cached for 10 seconds. With a 20s evaluation interval, each evaluation reads a value at most one tick stale.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
