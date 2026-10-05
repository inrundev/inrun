# deployments/

Live deployments. `public/` runs six runtimes in one cluster, each in its own namespace, aggregated by one Control Center at [cc.orkestra.sh](https://cc.orkestra.sh).

| Runtime | Operator | CRDs | Shows |
|---------|----------|------|-------|
| `orkestra-system-01` | hello-website | 1 | Deployment + Service from a `Website` CR (also hosts the Control Center) |
| `orkestra-system-02` | website-with-serviceaccount | 1 | The same, plus a ServiceAccount per instance |
| `orkestra-system-03` | secret-distribution | 1 | Copies a Secret across namespaces from a cluster-scoped CR |
| `orkestra-system-04` | app-platform | 5 | ReplicaSets, generated API keys with 90-day rotation, ConfigMap distribution |
| `orkestra-system-05` | data-platform | 10 | Ingestion to delivery, 30/60-day credential rotation |
| `orkestra-system-06` | network-suite | 7 | Traffic routing, self-signed TLS, 180-day key rotation |

Each `cluster-NN/` holds `katalog.yaml`, `crd.yaml` and `cr.yaml`. Runtimes 01 and 02 manage the same CRD and are split by `allowedNamespaces` (`demo-01`, `demo-02`); 04 to 06 own their own API groups.

```bash
cd deployments/public
make all          # or make cluster-01 … cluster-06
make clean
```

Each target creates the namespace, applies the CRD, applies the RBAC from `ork generate bundle`, installs via Helm and applies the CR.
