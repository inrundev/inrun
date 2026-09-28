# Operator Autoscaler
*A runtime‑native autoscaler for operatorBox:es.*

The Operator Autoscaler is a built‑in Orkestra subsystem that dynamically adjusts an operator’s **worker count**, **queue depth**, and **resync interval** based on **runtime metrics**, **time windows**, and **cron expressions**. It runs inside the same process as the operator and requires no external controllers, Deployments, or CRDs.

Autoscaling is fully declarative. You describe *when* an operator should scale and *how* it should scale. Orkestra evaluates these conditions at runtime and applies overrides instantly. When conditions are no longer met, Orkestra automatically restores the CRD’s declared baseline.

The autoscaler becomes the **source of truth** for operator performance, and the runtime trusts it to make these adjustments throughout the operator lifecycle.

---

## The autoscaler's inputs

Operators are control loops, not Pods. Their performance is measured in:

- queue pressure  
- worker utilization  
- reconcile duration  
- event throughput  

These signals are already tracked per CRD inside the operatorBox. The autoscaler reads them directly from memory — no additional instrumentation required.

---

## What the autoscaler can do

- Scale workers up or down  
- Increase or decrease queue depth  
- Adjust resync frequency  
- Trigger scaling based on:
  - metrics (`metrics.*`)
  - clock windows
  - day‑of‑week
  - cron expressions
  - OR/AND logic  
- Apply overrides immediately  
- Restore baseline automatically  
- Enforce cooldown to prevent oscillation  

All without restarting Orkestra.

---

## Documentation structure

This folder contains the full autoscaler documentation:

### 1. [Condition Engine](autoscaler-condition-engine.md)  
How autoscale conditions are evaluated, including metrics, time windows, cron expressions, and OR/AND semantics.

### 2. [Metrics](autoscaler-metrics.md)  
The autoscaler’s metric namespace (`metrics.*`), how metrics are collected, and what each metric means.

### 3. [Runtime Behavior](autoscaler-runtime-behaviour.md)  
How the autoscaler loop runs inside Kordinator, how overrides are applied, and how baseline restoration works.

### 4. [YAML Reference](autoscaler-yaml-reference.md)  
The complete schema for `operatorBox.autoscale`, including examples and field‑level explanations.

### 5. [Scenarios](scenarios.md)  
Real‑world autoscaling patterns: business hours, nightly batch windows, traffic‑based scaling, weekend scale‑down, and more.

---

## Mental model

Think of the autoscaler as a **mini‑controller** inside Orkestra:

- The **CRD** defines the steady‑state baseline.  
- The **autoscaler** decides when to override it.  
- **Kordinator** applies the decision live.  

This separation mirrors Kubernetes (Deployment → HPA → Kubelet), but applied to operators themselves.

---

## Try it

```bash
ork init --pack advanced
cd 12-autoscale/01-without-autoscaler
```

Follow the README — it starts with baseline behavior (no autoscaler) and adds progressively more complex autoscaling across five examples.