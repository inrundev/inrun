# pkg/gateway/handlers

HTTP handler constructors for the gateway's health-server routes.

| File | Routes | Purpose |
|------|--------|---------|
| `katalog.go` | `GET /katalog`, `GET /katalog/{crd}` | Gateway stats surface — admission, conversion, deletion-protection, namespace-protection counts per CRD. Fetched by the Control Center and merged with runtime `/katalog` data by GVR key. |

These are thin constructors — all business logic lives in `pkg/gateway/webhook`.
