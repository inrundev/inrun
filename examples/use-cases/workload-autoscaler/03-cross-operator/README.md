# 03 — Workload Autoscaler: Cross-Operator

Scale a Deployment based on a sibling operator's queue depth. JobWorker reads JobQueue's `status.queueDepth` via the `cross:` block and scales its Deployment on each reconcile.

---

## What You Learn

- How to wire `cross:` data into `autoscale:` conditions
- How two CRDs in one Katalog observe each other
- That any field a sibling publishes to status is a valid scaling signal
- The difference between this pattern and external API scaling (02): in-binary cross reads use the informer cache; external makes an HTTP call per resync

---

## Run the Example

### 1. Validate

```bash
ork validate
```

### 2. Simulate

```bash
ork simulate
```

### 3. Start Orkestra

```bash
ork run
```

Both the `jobqueue` and `jobworker` reconcilers start in the same process.

### 4. Start the Control Center

```bash
ork control
# username:password → orkestra
# → http://localhost:8081
```

You will see both operators. JobWorker starts at 2 replicas — queue depth (150) is already above the scale-up threshold.

### 5. Watch the worker scale up

```bash
kubectl get deployment my-worker -w
```

Within two resyncs (60s), the worker adds 2 replicas per tick until it reaches `max: 10` or the queue drains.

### 6. Drain the queue

```bash
kubectl patch jobqueue my-queue --type=merge -p '{"spec":{"initialDepth":0}}'
```

JobQueue's next reconcile sets `status.queueDepth = 0`. JobWorker reads the new depth on its next resync via `cross:` — depth is below `lessThan: "20"`, scale-down fires **after the cooldown**.

---

## How It Works

```yaml
# JobQueue publishes its depth to status
status:
  fields:
    - path: queueDepth
      value: "{{ .spec.initialDepth | default 0 }}"

# JobWorker reads it via observe.cross: and scales on it
operatorBox:
  observe:
    cross:
      - crd: jobqueue
        selector:
          name: "{{ .spec.queueName }}"
          namespace: "{{ .metadata.namespace }}"
        as: queue
  reconcile:
    onReconcile:
      deployments:
        - name: "{{ .metadata.name }}"
          autoscale:
            min: 2
            max: 10
            cooldown: 2m
            scaleUp:
              conditions:
                when:
                  - field: cross.queue.status.queueDepth
                    greaterThan: "100"
              increment: 2
            scaleDown:
              conditions:
                when:
                  - field: cross.queue.status.queueDepth
                    lessThan: "20"
              decrement: 1
```

`cross.queue.status.queueDepth` is the same field path syntax used in `when:` conditions elsewhere in the Katalog. In-binary cross lookups read from the informer cache; for cross-binary or cross-cluster, `observe.cross:` with a `source:` block resolves via HTTP.

In a real system, `queueDepth` would come from the queue operator's own metrics read (Kafka consumer lag, SQS ApproximateNumberOfMessages, etc.) written to its status on each reconcile.

---

## E2E

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
