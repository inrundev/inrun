# Environment Notes

Notes that reflect the runtime environment Orkestra is executing in. Values are determined once at startup and are constant for the lifetime of the process.

## Reference

| Note | Description |
|------|-------------|
| `inCluster` | Reports whether Orkestra is running inside a Kubernetes pod. |

## Examples

```yaml
# inCluster
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
