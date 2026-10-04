# pkg/gateway

The gateway is the external-facing component of Orkestra. It owns every concern that crosses the cluster boundary: serving CRs over HTTP, admission and conversion webhooks, TLS certificate management, and notification dispatch.

| Sub-package | Responsibility |
|-------------|----------------|
| [api/](api/README.md) | Gateway API — deliver CRs over HTTP without a kubeconfig |
| [webhook/](webhook/README.md) | Admission, conversion, deletion-protection, namespace-protection webhooks + housekeeper |
| [certmanager/](certmanager/README.md) | TLS certificate lifecycle for all webhook endpoints |
| [handlers/](handlers/README.md) | `/katalog` and `/katalog/{crd}` HTTP handler constructors |
