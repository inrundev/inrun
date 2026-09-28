# 34 — Environment Notes

Notes that reflect the runtime environment Orkestra is executing in. Values are determined once at startup and are constant for the lifetime of the process.

## Reference

### `inCluster`

Reports whether Orkestra is running inside a Kubernetes pod. Determined once at startup by checking for the service account token path. Use with `ternary` to write Katalogs that work both locally (via `ork run`) and deployed in-cluster without a separate file per environment.

Keywords: cluster, environment, pod, local, dev, remote, endpoint, boolean, inclu

```yaml
# Switch a remote reconciler endpoint between local dev and in-cluster service
notes:
  functions:
    - name: reconcilerEndpoint
      expression: '{{ ternary inCluster "http://webapp-reconciler.default.svc.cluster.local:8025/reconcile" "http://localhost:8025/reconcile" }}'

operatorBox:
  reconcile:
    default: false
    remote:
      endpoint: "{{ reconcilerEndpoint }}"
```

---

## Quick reference

| Note | Accepts | Returns | Use in |
|------|---------|---------|--------|
| `inCluster` | — | `bool` | `remote.endpoint`, `when` conditions, status fields |
