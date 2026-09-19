# Protocols

The `protocol:` field on an external call selects the wire protocol. The default (omitted) is HTTP. All protocols share the same Katalog syntax — only `url:`, `query:`, and `auth:` differ per protocol.

```yaml
onReconcile:
  external:
    - name: myCall
      protocol: redis        # or postgres, mongo, kafka, prometheus
      url: "..."
      query: "..."
      auth:
        secretRef:
          namespace: default
          name: my-secret
          key: password
      cacheFor: 30s
      continueOnError: true
```

Results are available in every template expression under `.external.<name>.*`.

!!! tip "Orkestra scales any Deployment"
    You do not need to own a Deployment for Orkestra to scale it. Add it by name and namespace to `deployments:`, or use `apiTypes.kind: Deployment` with a label selector to manage a fleet cluster-wide. Protocol results drive `autoscale:` conditions the same way regardless of whether the Deployment was created by the katalog or already existed.

---

## Supported protocols

### `http` (default)

General-purpose HTTP/HTTPS call. Supports GET, POST, PUT, PATCH, DELETE. JSON responses are auto-parsed — keys are merged directly into the result map.

```yaml
- name: health
  url: "{{ .spec.serviceUrl }}/health"
  method: GET
  timeout: 5s
```

**Result:** `status`, `body`, `error`, `called` + all JSON keys from the response body.

---

### `prometheus`

Prometheus instant query via the HTTP API (`/api/v1/query`). Accepts full PromQL — aggregations, rates, label selectors, subqueries. Points at any Prometheus-compatible endpoint (Prometheus, Thanos, VictoriaMetrics, Grafana Mimir, Orkestra's own `/api/v1/query`).

```yaml
- name: errorRate
  protocol: prometheus
  url: "http://prometheus.monitoring.svc:9090"
  query: "rate(http_requests_total{status=~\"5..\"}[5m])"
  cacheFor: 30s
  continueOnError: true
```

**Result:** `result` (first series value), `raw` (full Prometheus response), `error`, `called`.

**Notes:** `promValue`, `promSum`, `promMax`, `promAboveThreshold`, `promBelowThreshold`, `promSeriesCount`, `promLabelValues`.

---

### `redis`

Executes a single Redis command. `query:` is the command and its arguments. The credential from `auth:` is injected as the Redis password.

```yaml
- name: queueDepth
  protocol: redis
  url: "redis://redis-svc.default.svc:6379"
  query: "LLEN my-queue"
  auth:
    secretRef:
      namespace: default
      name: redis-secret
      key: password
```

**`query:` examples:** `GET mykey`, `HGET myhash field`, `LLEN mylist`, `SCARD myset`, `DBSIZE`, `PING`

**Result:** `result` (string value of the response), `raw.value` (untyped), `error`, `called`.

---

### `postgres`

Executes a SQL query against a PostgreSQL database. `query:` is a SQL string. The credential from `auth:` is injected as the password, overriding any password in the URL.

```yaml
- name: userCount
  protocol: postgres
  url: "postgres://app@pg.default.svc:5432/mydb"
  query: "SELECT COUNT(*) FROM users WHERE active = true"
  auth:
    secretRef:
      namespace: default
      name: pg-secret
      key: password
  cacheFor: 60s
```

**Result:** `result` (first column of first row), `rows` (`[]map[string]interface{}`), `rowCount`, `error`, `called`.

---

### `mongo`

Executes a MongoDB find query. `query:` is `"database.collection"` with an optional JSON filter.

```yaml
- name: pendingOrders
  protocol: mongo
  url: "mongodb://mongo.default.svc:27017"
  query: 'orders.items {"status": "pending"}'
  auth:
    secretRef:
      namespace: default
      name: mongo-secret
      key: password
```

**`query:` syntax:**
- `"mydb.mycollection"` — find all documents
- `"mydb.mycollection {\"field\": \"value\"}"` — find with filter

**Result:** `result` (JSON string of first document), `documents` (`[]map[string]interface{}`), `count`, `error`, `called`.

---

### `kafka`

Reads consumer group lag or topic metadata. Read-only — never produces messages.

```yaml
- name: queueLag
  protocol: kafka
  url: "kafka-broker.default.svc:9092"
  query: "my-consumer-group/my-topic"
  continueOnError: true
```

**`query:` syntax:**
- `"group/topic"` — consumer group lag for a topic (total lag across all partitions)
- `"@topic"` — topic metadata: partition count

**Result (`group/topic`):** `result` (total lag), `lag`, `partitions`, `error`, `called`.

**Result (`@topic`):** `result` (partition count), `partitionCount`, `error`, `called`.

**Auth (SASL/PLAIN):**

Store `username:password` in a Kubernetes Secret and reference it via `auth.secretRef`:

```yaml
- name: queueLag
  protocol: kafka
  url: "kafka-broker.default.svc:9092"
  query: "my-consumer-group/my-topic"
  continueOnError: true
  auth:
    secretRef:
      name: kafka-credentials
      namespace: default
      key: sasl
```

Secret:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kafka-credentials
  namespace: default
stringData:
  sasl: "myuser:mypassword"
```

The credential is used for SASL/PLAIN. Unauthenticated clusters work without `auth:`. SCRAM-SHA-256/512 is not yet supported.

**Autoscaler pattern:**
```yaml
- name: lag
  protocol: kafka
  url: "{{ .spec.kafkaBrokers }}"
  query: "workers/jobs"
  cacheFor: 15s
  continueOnError: true
```
```yaml
autoscale:
  scaleUp:
    conditions:
      when:
        - field: "{{ .external.lag.lag }}"
          greaterThan: "10000"
    increment: 2
```

**Validation rule pattern:**
```yaml
- name: topicCheck
  protocol: kafka
  url: "{{ .spec.kafkaBrokers }}"
  query: "@{{ .spec.topic }}"
  continueOnError: true
  timeout: 5s
```
```yaml
admission:
  validation:
    rules:
      - name: topic exists
        value: "{{ .external.topicCheck.error }}"
        equals: ""
        message: "topic {{ .spec.topic }} does not exist or the broker is unreachable"
      - name: topic has enough partitions
        value: "{{ .external.topicCheck.partitionCount }}"
        greaterThan: "0"
        message: "topic {{ .spec.topic }} must have at least one partition"
```

---

## Validation rules

Any protocol can gate admission. The pattern is the same — fetch with `continueOnError: true`, then assert on the result fields in `validation.rules`.

**Postgres — connection count gate:**
```yaml
onReconcile:
  external:
    - name: pgHealth
      protocol: postgres
      url: "{{ .spec.databaseURL }}"
      query: "SELECT count(*) AS active FROM pg_stat_activity WHERE state = 'active'"
      continueOnError: true
      timeout: 5s
```
```yaml
admission:
  validation:
    rules:
      - name: postgres is reachable
        value: "{{ .external.pgHealth.error }}"
        equals: ""
        message: "cannot reach database: {{ .external.pgHealth.error }}"
      - name: connection count is sane
        value: "{{ .external.pgHealth.active }}"
        lessThan: "50"
        message: "database is overloaded ({{ .external.pgHealth.active }} active connections)"
```

**Redis — queue depth gate:**
```yaml
onReconcile:
  external:
    - name: queueDepth
      protocol: redis
      url: "{{ .spec.redisURL }}"
      query: "LLEN jobs"
      continueOnError: true
```
```yaml
admission:
  validation:
    rules:
      - name: queue is not backed up
        value: "{{ .external.queueDepth.result }}"
        lessThan: "1000"
        message: "job queue is too large ({{ .external.queueDepth.result }} items) — retry later"
```

**Prometheus — error rate SLO gate:**
```yaml
onCreate:
  external:
    - name: errorRate
      protocol: prometheus
      url: "http://prometheus.monitoring.svc:9090"
      query: 'rate(http_requests_total{job="{{ .spec.service }}",status=~"5.."}[5m])'
      continueOnError: true
```
```yaml
admission:
  validation:
    rules:
      - name: error rate is acceptable
        value: "{{ .external.errorRate.error }}"
        equals: ""
        message: "prometheus unreachable: {{ .external.errorRate.error }}"
      - name: SLO is healthy
        value: "{{ promValue .external.errorRate }}"
        lessThan: "0.01"
        message: "error rate {{ promValue .external.errorRate | printf \"%.4f\" }} exceeds 1% SLO"
```

---

## In development

These protocols are declared in the type system and pass `ork validate`, but their clients are not yet implemented. The runner falls through to HTTP for unknown protocols — using them before implementation will not produce useful results.

### `grpc`

Unary gRPC call. `query:` will be the fully-qualified method name (`package.Service/Method`). The request body will come from `body:`. Planned for a future release.

### `nats`

NATS KV read or JetStream stream info. `query:` syntax: `"bucket.key"` for KV, stream name for stream metadata. Planned for a future release.

### `mqtt`

Reads a retained MQTT topic. `query:` is the topic path. Planned for a future release.
