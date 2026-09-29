# Scaling Signals

Orkestra can scale any Deployment — not just ones it created. Add a Deployment by name to `onReconcile:` and Orkestra manages its replica count. Use `apiTypes.kind: Deployment` to watch every Deployment in the cluster and scope with label selectors.

`autoscale:` conditions evaluate against the same Resolver data map used everywhere else in a Katalog. Any source resolved before deployment evaluation — `external:`, `cross:`, time notes, user-defined notes — is available as a field reference in `when:` and `or:` blocks.

---

## Cluster-wide scaling

`apiTypes.kind: Deployment` makes Orkestra watch every Deployment as a virtual CR. Three mechanisms scope which ones are managed — use any combination:

**Whitelist — only watch specific namespaces:**
```yaml
apiTypes:
  deployment:
    kind: Deployment
    labelSelector:
      app.kubernetes.io/managed-by: my-platform
    operatorBox:
      runtime:
        allowedNamespaces:
          - default
          - my-platform
```

**Blacklist — watch everything except these namespaces:**
```yaml
apiTypes:
  deployment:
    kind: Deployment
    labelSelector:
      app.kubernetes.io/managed-by: my-platform
    operatorBox:
      runtime:
        restrictedNamespaces:
          - kube-system
          - kube-public
```

```yaml
onReconcile:
  deployments:
    - name: "{{ .metadata.name }}"
      namespace: "{{ .metadata.namespace }}"
      autoscale:
        scaleUp:
          conditions:
            when:
              - field: "{{ promAboveThreshold .external.queueDepth 10000 }}"
                equals: "true"
          increment: 2
        scaleDown:
          conditions:
            when:
              - field: "{{ promBelowThreshold .external.queueDepth 500 }}"
                equals: "true"
          target: 2
```

`allowedNamespaces` and `restrictedNamespaces` are mutually exclusive — `ork validate` rejects both being set on the same CRD. `labelSelector` can be combined with either. Every Deployment that passes the scope is reconciled by the same autoscale policy; the `external:` block resolves once per Deployment per reconcile.

Namespace protection is enforced at admission time by the gateway. When `security.namespaceProtection.enabled: true` is declared in the Katalog (requires `gateway.enabled: true` — the gateway registers the ValidatingWebhookConfiguration), any CR targeting a restricted namespace is rejected at admission before the reconciler runs.

```yaml
gateway:
  endpoint: http://orkestra-gateway:8080

security:
  namespaceProtection:
    enabled: true
    failurePolicy: Fail
```

---

## Time conditions

```yaml
scaleUp:
  conditions:
    when:
      - dayOfWeek:
          weekday: true
      - time:
          after: "09:00"
          before: "18:00"
  target: 8

scaleDown:
  conditions:
    when:
      - field: "{{ inBusinessHours }}"
        equals: "false"
  target: 2
```

Time conditions use the built-in `time:` and `dayOfWeek:` blocks, or user-defined notes that wrap them. The reconciler re-evaluates on every resync and converges to the correct replica count for the current time.

---

## External HTTP

```yaml
onReconcile:
  external:
    - name: queue
      url: "{{ .spec.metricsEndpoint }}"
      timeout: 5s
      continueOnError: true

  deployments:
    - name: "{{ .metadata.name }}"
      autoscale:
        scaleUp:
          conditions:
            when:
              - field: external.queue.pendingJobs
                greaterThan: "100"
          increment: 2
```

`external:` resolves before deployment evaluation. The response is available at `external.<name>.*` when conditions are checked. Any JSON HTTP endpoint works: Kafka consumer lag, RabbitMQ queue depth, SQS queue attributes, Prometheus instant queries, custom APIs. When the response body is a valid JSON object, its keys are merged directly into the result — `external.queue.pendingJobs` works for `{ "pendingJobs": 100 }`, and `external.queue.metrics.pendingJobs` works for `{ "metrics": { "pendingJobs": 100 } }`.

`continueOnError: true` means a temporarily unavailable endpoint does not error the reconcile — conditions are not met, the Deployment holds its current replica count, and the operator retries on the next resync.

The `deployments:` entry does not need to be a Deployment the katalog created. Add any Deployment by name and namespace — Orkestra takes it under management.

!!! tip "Reactive Policy Engine"
    For native protocol clients — Prometheus, Redis, Kafka, Postgres, MongoDB — see [External Protocols](../operatorbox/07-external/04-protocols.md). Each protocol returns a flat result map and can drive `when:` conditions the same way as HTTP.

---

## Prometheus metrics

`protocol: prometheus` makes any PromQL expression a first-class scaling signal — no Prometheus adapter, no KEDA, no HPA custom metrics pipeline required.

```yaml
onReconcile:
  external:
    - name: queueDepth
      protocol: prometheus
      url: "http://prometheus.monitoring.svc:9090"
      query: "sum(rabbitmq_queue_messages{vhost=\"/\"})"
      cacheFor: 15s
      continueOnError: true

    - name: errorRate
      protocol: prometheus
      url: "http://prometheus.monitoring.svc:9090"
      query: "rate(http_requests_total{job=\"api\",status=~\"5..\"}[5m])"
      cacheFor: 30s
      continueOnError: true

deployments:
  - name: "{{ .metadata.name }}"
    autoscale:
      scaleUp:
        conditions:
          when:
            - field: "{{ promAboveThreshold .external.queueDepth 10000 }}"
              equals: "true"
            - field: "{{ promBelowThreshold .external.errorRate 0.05 }}"
              equals: "true"
        increment: 2
      scaleDown:
        conditions:
          when:
            - field: "{{ promBelowThreshold .external.queueDepth 500 }}"
              equals: "true"
        target: 2
```

`cacheFor:` avoids a Prometheus round-trip on every reconcile while keeping the signal fresh. `continueOnError: true` means a Prometheus outage holds the current replica count rather than erroring the reconcile.

The full PromQL expression language is available — aggregations, rates, label selectors, subqueries, histogram quantiles. Any metric in Prometheus can drive scale-up or scale-down.

```yaml
# p99 latency above SLO — scale up
- name: p99Latency
  protocol: prometheus
  url: "http://prometheus.monitoring.svc:9090"
  query: "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{job=\"api\"}[5m])))"
  cacheFor: 30s
  continueOnError: true
```

```yaml
autoscale:
  scaleUp:
    conditions:
      when:
        - field: "{{ promAboveThreshold .external.p99Latency 0.5 }}"
          equals: "true"
    increment: 1
```

---

## Cross-operator metrics

```yaml
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
            scaleUp:
              conditions:
                when:
                  - field: cross.queue.status.queueDepth
                    greaterThan: "100"
              increment: 2
```

`observe.cross:` reads a sibling CRD's resolved state. In-binary lookups go through the informer cache. For cross-binary or cross-cluster, add a `source:` block to fetch via HTTP with caching. Any field the sibling publishes to status is reachable at `cross.<name>.status.*`.

---

## Notes as scaling guards

```yaml
scaleUp:
  conditions:
    when:
      - field: "{{ hasCrashingPod .children.deployment }}"
        equals: "false"
      - field: external.queue.pendingJobs
        greaterThan: "100"
  increment: 2
```

Notes can guard scale-up — here, the operator only scales when there are no crashing pods and the queue is deep. Notes that return booleans evaluate naturally in `when:` blocks.

---

## Composite conditions

```yaml
scaleUp:
  conditions:
    or:
      - field: external.kafka.consumerLag
        greaterThan: "1000"
      - field: cross.worker.metrics.queueDepth
        greaterThan: "200"
      - time:
          after: "08:00"
          before: "09:00"
        dayOfWeek:
          weekday: true
```

`or:` evaluates as OR — scale up if any condition passes. `when:` evaluates as AND — all conditions must pass. Both can appear in the same `conditions:` block; `when:` and `or:` are evaluated independently and combined with AND.
