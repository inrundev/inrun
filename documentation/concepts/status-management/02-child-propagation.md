# Child Resource Propagation

After reconcile templates create child resources, Orkestra reads them back and makes their status available in the template resolver under the `children` key. Status fields can then reference the live state of child resources.

```yaml
operatorBox:
  emit:
    status:
      fields:
        - path: readyReplicas
          value: "{{ get .children.deployment \"status\" \"readyReplicas\" }}"

        - path: availableReplicas
          value: "{{ get .children.deployment \"status\" \"availableReplicas\" }}"

        - path: loadBalancerIP
          value: "{{ serviceLoadBalancerIP .children.service }}"
```

---

## Access patterns

`.children.deployment` — the first (or only) Deployment created by this CR. The ergonomic path for single-Deployment operators.

`{{ readyReplicas (index .children.deployments "my-site-api") }}` — access by exact Kubernetes name when multiple Deployments exist.

The same patterns work for all resource types:

| Singular | Plural |
|---|---|
| `.children.deployment` | `.children.deployments` |
| `.children.service` | `.children.services` |
| `.children.secret` | `.children.secrets` |
| `.children.configMap` | `.children.configMaps` |
| `.children.job` | `.children.jobs` |
| `.children.cronJob` | `.children.cronJobs` |
| `.children.pod` | `.children.pods` |
| `.children.serviceAccount` | `.children.serviceAccounts` |

---

## First reconcile behaviour

When a CR is first created, the Deployment exists but Kubernetes has not yet populated `status.readyReplicas` — it is zero or absent. Missing fields resolve to `""`. The status field writes an empty string. On the next reconcile cycle (triggered by the Deployment's status change event), the value is populated. This is expected eventual consistency.

---

## API calls per reconcile

!!! note "API calls per reconcile"
    Layer 3 makes one GET per child resource declared in the Katalog — bounded by the number of template declarations, not by the number of CRs. An operator with one Deployment and one Service makes two GETs per reconcile. These are reads against the API server, not the informer cache, because child resource informers are not registered by default.
