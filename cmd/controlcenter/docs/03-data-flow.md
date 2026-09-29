# 03 — Data Flow

Data travels in one direction: Orkestra runtime (+ optional gateway) → `Client` → `ControlCenter` in-memory state → view-model struct → Go HTML template → browser.

## Orkestra runtime API endpoints

The Control Center calls these endpoints on each connected Orkestra runtime:

| Endpoint | Go type | Used by |
|----------|---------|---------|
| `GET /katalog` | `KatalogResponse` | Background fetch loop (every instance, every tick) |
| `GET /katalog/{crd}/health` | `CRDHealth` | `handleCRDDetail`, `handleCRDDocs` |
| `GET /katalog/{crd}` | `CRDInfo` | `handleCRDDetail`, `handleCRDDocs` |
| `GET /katalog/{crd}/cr` | `CRListResponse` | `handleCRList` |
| `GET /katalog/{crd}/cr/{ns}/{name}` | `CRDetailResponse` | `handleCRDetail` |
| `GET /katalog/{crd}/cr/{ns}/{name}/events` | `CREventsResponse` | `handleCRDetail` |
| `GET /katalog/raw` | raw JSON | `/katalog/{name}/raw` proxy |
| `GET /katalog/{crd}/raw` | raw JSON | `/katalog/{name}/crd/{crd}/raw` proxy |

All calls go through `cc/client.go`. The generic helper `getJSON[T]` handles JSON decoding and error wrapping.

## Gateway API endpoints

When a runtime advertises a companion gateway via `"gatewayEndpoint"` in its `/katalog` response, the Control Center also queries the gateway for webhook stats and (when Serve is enabled) proxies schema and apply requests:

| Endpoint | Go type | Used by |
|----------|---------|---------|
| `GET /katalog/{crd}` | `GatewayCRDStats` | `handleCRDDetail` — merges admission, conversion, deletion/namespace protection stats |
| `GET /api/v1/schema?target={target}` | `SchemaResponse` (flat field map) | `fetchIDPFields` (form render); `handleIDPSchema` proxy |
| `POST /api/v1/apply` | `ApplyResponse` | `handleIDPApplyForm` (form submit); `handleIDPApply` proxy |

The gateway URL is stored on `Instance.GatewayEndpoint` when the runtime katalog is fetched. Webhook stats are queried on-demand at CRD detail page load. Schema fetch and apply for the `[+ Create]` form happen server-side in `handleIDPCreateForm`/`handleIDPApplyForm` — the CC adds the `Authorization: Bearer` header so the token stays server-side. The `/api/idp/*` routes are a separate, standalone proxy pair for callers that want raw gateway JSON directly (see [02-routing.md](02-routing.md)).

## IDP mode

When the runtime includes `"serveEnabled": true` and a non-empty `"target"` on a `CRDSummaryResponse` entry in the `/katalog` response, the CR list page for that CRD renders a `[+ Create]` button linking to `/katalog/{kat}/crd/{crd}/cr/create`. That route fetches the flat field schema for `Target` (`GET /api/v1/schema?target=...`) and renders `idp_form.html`. On submit, the browser POSTs a flat `{"target": "...", ...fields}` object back to the same route; `handleIDPApplyForm` forwards it to the gateway's `POST /api/v1/apply`, which builds the CR. Control Center never constructs a CR itself — see [06-idp-form.md](06-idp-form.md).

The `GATEWAY_TOKEN` env var on the CC is the bearer token sent to the gateway. The browser only ever talks to the CC — no cross-origin requests, no token exposure.

```
KatalogResponse.CRDs[].IdpEnabled  ← per-CRD flag from runtime /katalog
KatalogResponse.CRDs[].Target      ← serve.target (or lowercased kind); empty when Serve disabled
KatalogResponse.GatewayEndpoint    ← stored on Instance; used by CC proxy handlers
ControlCenter.gatewayToken         ← from GATEWAY_TOKEN env var; never sent to browser
```

## Periodic fetch

`FetchKatalog` is called for every instance on every tick. The result is conditionally stored on `inst.Katalog` — see the cache-update rule below.

```
KatalogResponse
  .Name, .Description, .Version, .Author, .License
  .Healthy, .OrkReady, .DegradedReason
  .IsKonductor                    ← true only on the pod holding the leader election lease
  .CRDs            []CRDSummary   ← high-level health per CRD
  .StatusCounts                   ← healthy/started/pending/degraded counts
  .RuntimeVersion                 ← version string from the runtime
  .GatewayEndpoint                ← base URL of companion gateway, empty if none
```

### Cache-update rule

Three cases on each fetch:

| Response | `inst.Katalog` state | Action |
|----------|----------------------|--------|
| `isKonductor: true` | any | Replace — leader data is authoritative |
| `isKonductor: false` | nil | Accept — no data yet, something beats nothing; CRDs may show as pending until a leader response arrives |
| `isKonductor: false` | non-nil | Discard — keep last known-good snapshot; prevents flipping between healthy and pending |

This matters when `replicaCount > 1`: the Kubernetes Service round-robins to any pod. Only the leader sets `isKonductor: true` (via `DependencyKordinator.Kordinate()`); follower pods never run reconcilers so their CRD health stays "pending". The second case ensures all katalogs appear immediately on startup; the third case ensures the display does not flip once stable.

The `CRDSummary` slice is what drives the Katalog panel and the index. It does not include deep detail — for that, the user navigates to a CRD page, which triggers a fresh `FetchCRDDetail` call at request time.

## On-demand fetch

`FetchCRDDetail` merges two runtime calls — `GET /katalog/{crd}/health` and `GET /katalog/{crd}` — into a single `*CRDDetail`. When a gateway endpoint is configured, `handleCRDDetail` additionally calls `FetchGatewayCRDStats` and merges the webhook stats fields:

```
CRDDetail
  ← CRDHealth:        state, workers, queue depth, error counts, dependencies
  ← CRDInfo:          GVK, GVR, scope, mode, workers config, RBAC
  ← GatewayCRDStats:  Admission, Conversion, DeletionProtection, NamespaceProtection
                      (only when GatewayEndpoint is set; runtime fields take lower precedence)
```

If the runtime is unreachable, `handleCRDDetail` renders a degraded view with `State: "offline"` rather than returning an error page. Gateway fetch failures are soft — the page renders with whatever runtime data is available.

## View models

Each handler builds a typed view-model struct and passes it to `renderTemplate`. The struct name matches the template name:

| Handler | View-model | Template |
|---------|-----------|----------|
| `handleIndex` | `IndexData` | `index.html` |
| `handleKatalogPanel` | `KatalogData` | `katalog.html` |
| `handleCRDDetail` | `map[string]interface{}` | `crd.html` |
| `handleCRList` | `CRListView` | `cr_list.html` |
| `handleCRDetail` | `CRDetailView` | `cr_detail.html` |
| `handleDocsLanding` | `DocsLandingData` | `docs.html` |
| `handleCRDDocs` | `CRDDocsData` | `crd_docs.html` |

`CRDDocsData` adds `Has*` boolean flags (`HasAdmission`, `HasConversion`, `HasProtection`, `HasRBAC`, `HasAutoscaler`, `HasRollback`) computed from the fetched `*CRDDetail`. Templates use these to conditionally render sections so missing features produce no empty headers.

## Snapshot API

`GET /api/snapshot` returns a lightweight JSON summary (no per-CRD detail) used by the index page JS to do partial DOM updates after each SSE `update` event. This avoids a full page reload while keeping stats current.

→ Next: [04-ui-design.md](04-ui-design.md)
