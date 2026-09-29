# Autoscale Profile

An autoscale profile is a named preset that expands into a complete `operatorBox.autoscale` block at Katalog load time.

Profiles are **relative** — they use the CRD's declared `workers` and `queue.maxDepth` as a baseline, then compute worker overrides, queue overrides, trigger thresholds, and timing from that baseline.

---

## Profiles

| Profile | Trigger | Workers override | Queue override | Interval | Cooldown |
|---------|---------|-----------------|----------------|----------|----------|
| `burst` | queueDepth > 60%  | baseline × 4 | baseline × 10 | 5s | 30s |
| `steady` | queueDepth > 40% AND workersBusy > 70% | baseline × 2 | baseline × 3 | 30s | 2m |
| `batch` | cron window 23:00 → 02:00 | baseline × 3 | baseline × 8 | 60s | 5m |
| `latency-sensitive` | P95 reconcile > 200ms | ⌈baseline × 2.5⌉ | — | 15s | 1m |
| `cost-optimized` | workersIdle > 60% AND queueDepth > 80% | max(1, baseline × 0.5) | baseline × 0.5 | 30s | 10m |

---

## Usage

```yaml
operatorBox:
  runtime:
    autoscale:
      profile: steady
  workers: 4
  queue:
    maxDepth: 100
```

For `steady` with the above baseline (workers=4, queueDepth=100):

- override workers = 4 × 2 = **8**
- override queueDepth = 100 × 3 = **300**
- trigger threshold = 300 × 40% = **120**
- trigger condition: `queueDepth > 120 AND workersBusyPercent > 70`

---

## Rules

**Profile or explicit — not both.**

```yaml
# Valid
autoscale:
  profile: burst

# Valid
autoscale:
  interval: 10s
  cooldown: 1m
  conditions:
    when:
      - field: metrics.queueDepth
        greaterThan: "500"
  do:
    workers: 16

# Invalid — rejected at load time
autoscale:
  profile: burst
  interval: 10s
```

**Unknown profiles fail fast.** An unrecognized name is a Katalog load error.

---

## Profile details

### `burst`

React instantly to spikes. Aggressive scaling, short interval, short cooldown.

```text
trigger:  queueDepth > 60% 
override: workers × 4, queueDepth × 10
timing:   interval 5s, cooldown 30s
```

Use when your operator handles unpredictable bursts and you need it to scale out before the queue fills.

### `steady`

Smooth, predictable scaling. Requires both queue pressure and worker saturation before scaling.

```text
trigger:  queueDepth > 40%  AND workersBusyPercent > 70
override: workers × 2, queueDepth × 3
timing:   interval 30s, cooldown 2m
```

Use for production operators with consistent load patterns.

### `batch`

Scale for a nightly processing window. Time-triggered, not load-triggered.

```text
trigger:  cron "0 23 * * *", duration 3h
override: workers × 3, queueDepth × 8
timing:   interval 60s, cooldown 5m
```

Use when you have a known batch window (nightly jobs, end-of-day processing).

### `latency-sensitive`

Keep reconcile latency low. Triggered by P95 latency, not queue depth.

```text
trigger:  reconcileDurationP95Ms > 200
override: workers × 2.5 (ceiling)
timing:   interval 15s, cooldown 1m
```

Use when users are sensitive to reconcile delay and you want to scale out before latency compounds.

### `cost-optimized`

Minimize resource usage during low activity. Scale down aggressively when idle.

```text
trigger:  workersIdlePercent > 60 AND queueDepth > 80% 
override: max(1, workers × 0.5), queueDepth × 0.5
timing:   interval 30s, cooldown 10m
```

Use for operators with long idle periods between bursts, where you want to give back resources between processing windows.

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| Unpredictable spikes, must not queue-block | `burst` |
| Consistent throughput, standard production | `steady` |
| Known batch window (nightly, weekly) | `batch` |
| Latency SLO, users feel slow reconciles | `latency-sensitive` |
| Long idle periods, cost matters | `cost-optimized` |

If none fit, write the autoscale block manually.
