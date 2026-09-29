# The Baseline

`00-controller-runtime-baseline` is the starting point. A standard controller-runtime operator for a WebApp CRD — the same structure kubebuilder generates.

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/00-controller-runtime-baseline
```

---

## What it looks like

```text
00-controller-runtime-baseline/
  api/v1alpha1/
    webapp_types.go          CRD type definitions
    zz_generated.deepcopy.go generated DeepCopy methods
  controller/
    webapp_controller.go     Reconcile method and sub-methods
  main.go                    Manager setup, scheme registration, leader election
  Dockerfile                 Multi-stage Go build → distroless image
  chart/                     Helm chart for production deployment
    Chart.yaml
    values.yaml
    templates/
      deployment.yaml
      clusterrole.yaml
      clusterrolebinding.yaml
      serviceaccount.yaml
```

---

## The machinery you write

The Reconcile method itself is 15 lines:

```go
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    webapp := &demov1alpha1.WebApp{}
    if err := r.Get(ctx, req.NamespacedName, webapp); err != nil {
        if errors.IsNotFound(err) { return ctrl.Result{}, nil }
        return ctrl.Result{}, err
    }
    if err := r.reconcileDeployment(ctx, webapp); err != nil { return ctrl.Result{}, err }
    if err := r.reconcileService(ctx, webapp); err != nil { return ctrl.Result{}, err }
    webapp.Status.Phase = "Running"
    if err := r.Status().Update(ctx, webapp); err != nil { return ctrl.Result{}, err }
    return ctrl.Result{}, nil
}
```

What surrounds it:

| File | Purpose | Lines |
|------|---------|-------|
| `webapp_types.go` | CRD types, `GroupVersionKind`, `AddToScheme` | ~60 |
| `zz_generated.deepcopy.go` | Generated — never manually edited | ~50 |
| `webapp_controller.go` | Reconcile + 2 sub-methods | ~120 |
| `main.go` | Manager, scheme, leader election, health server | ~50 |
| `Dockerfile` | Multi-stage build | ~20 |
| `chart/` | Helm chart, 4 templates | ~150 |

The reconcile logic is in `webapp_controller.go`. Everything else is machinery.

---

## What you own

- The informer — from `ctrl.NewControllerManagedBy(...).Owns(...)` 
- The workqueue — implicit in the manager
- The worker — single goroutine by default, no control over pool size
- Leader election — configured in `main.go`
- Health endpoints — `healthz`, `readyz` registered in `main.go`
- Metrics — basic controller metrics exposed via the manager's metrics server; custom metrics instrumented by you
- Panic recovery — not included; a panic kills the controller
- The Helm chart — deployed, upgraded, and rolled back by you

---

## Running it

```bash
# Install CRDs
kubectl apply -f config/crd/bases/

# Start locally
go run ./main.go

# Apply a CR
kubectl apply -f cr.yaml
kubectl get webapps
kubectl get deployments
kubectl get services
```

---

## Why this matters as a starting point

Every subsequent option removes some of this machinery. The baseline is here so you can measure what you actually gave up — and what you got back.

→ [02 — Declarative](./02-declarative.md) — the same operator as a pure Katalog. No Go, no binary.
