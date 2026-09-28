# Extending Orkestra with Provider Libraries

Providers extend Orkestra's declarative layer to external systems — AWS,
MongoDB, Stripe, Vault, or any API your operator needs to call. This document
explains how the provider dispatch works internally and how to write, register,
and test a new provider.

> [!WARNING]
> The provider system is still in active development. It's already wired into
> the runtime and the providers listed under [Current providers](#current-providers)
> are functional, but the interface, registration flow, and publishing format
> are not yet stable and may change without a deprecation period. Not
> recommended for production use yet.

---

## How providers flow through the runtime

The complete call chain from Katalog YAML to external API:

```
Katalog YAML
  └── providers:
        └── aws: [...]           ← block name + declarations

konstructRuntime (internal/construct.go)
  └── loadProviders(ctx)         ← builds the ProviderRegistry
  └── factory closure            ← captures providerRegistry
        └── NewGeneric Reconciler(..., providerRegistry)

DependencyKordinator
  └── startCRDWorkers
        └── entry.ReconcilerFactory()   ← calls the closure
              └── Generic Reconciler

Generic Reconciler.reconcileImpl
  └── runTemplateReconcile
        └── runProviders(ctx, obj, resolver, blocks, registry, kube)
              └── registry.Get("aws")          ← lookup by block name
              └── resolveProviderBlock(...)     ← template expressions resolved
              └── filterProviderDeclarations(...) ← when: conditions evaluated
              └── provider.Reconcile(ctx, req) ← your code runs here

On CR deletion (finalizer):
  └── runProviderDelete(ctx, obj, resolver, blocks, registry, kube)
        └── provider.Delete(ctx, req)
```

**Key insight:** The `ProviderRegistry` is captured in the factory closure in
`konstructRuntime`. It is declared before the factory loop so all CRD factories
capture the same fully-initialised registry. The registry never passes through
`DependencyKordinator` or `ReconcilerFactory` — the closure handles it.

---

## The provider interface

```go
// pkg/types/provider.go

type Provider interface {
    // Name returns the YAML block key: "aws", "mongodb", "stripe"
    Name() string

    // Reconcile is called after Kubernetes resources are reconciled.
    // Must be idempotent — called on every resync cycle, not just on create.
    Reconcile(ctx context.Context, req ReconcileRequest) error

    // Delete is called during finalizer execution.
    // Must clean up all external resources created for this CR.
    // Return nil if the resource does not exist — idempotent success.
    Delete(ctx context.Context, req DeleteRequest) error
}
```

`ReconcileRequest` carries everything your provider needs:

```go
type ReconcileRequest struct {
    Object         map[string]interface{}     // full CR: spec, status, metadata, children
    Declarations   []ProviderDeclaration      // resolved declarations (template expressions evaluated)
    Kube           KubeReader                 // read Secrets and ConfigMaps
    Logger         zerolog.Logger             // pre-tagged with crd, resource, request_id
    OwnerName      string                     // CR metadata.name
    OwnerNamespace string                     // CR metadata.namespace
}
```

`ProviderDeclaration` is one resolved item from the YAML block:

```go
type ProviderDeclaration struct {
    Kind   string            // "s3", "rds", "database", "user"
    Fields map[string]string // all field values, template expressions already resolved
}

// Convenience methods on ProviderDeclaration:
decl.Field("bucket", "default-bucket")  // value or default if absent/empty
decl.Require("instanceClass")           // value or error if absent/empty
```

---

## Writing a new provider

### Step 1: Create the package

```
pkg/providers/
  └── myprovider/
        └── provider.go
        └── provider_test.go
```

### Step 2: Implement the interface

```go
package myprovider

import (
    "context"
    "fmt"

    orktypes "github.com/orkspace/orkestra/pkg/types"
)

type Provider struct {
    client *MyExternalClient
}

func New(client *MyExternalClient) *Provider {
    return &Provider{client: client}
}

// NewFromEnv builds a provider from environment variables.
// Preferred over accepting raw credentials in arguments.
func NewFromEnv() (*Provider, error) {
    apiKey := os.Getenv("MY_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("myprovider: MY_API_KEY not set")
    }
    client := newClient(apiKey)
    return New(client), nil
}

func (p *Provider) Name() string { return "myprovider" }

func (p *Provider) Reconcile(ctx context.Context, req orktypes.ReconcileRequest) error {
    for _, decl := range req.Declarations {
        switch decl.Kind {
        case "widget":
            if err := p.reconcileWidget(ctx, req, decl); err != nil {
                return fmt.Errorf("widget: %w", err)
            }
        default:
            req.Logger.Warn().Str("kind", decl.Kind).Msg("myprovider: unknown kind — skipped")
        }
    }
    return nil
}

func (p *Provider) Delete(ctx context.Context, req orktypes.DeleteRequest) error {
    for _, decl := range req.Declarations {
        if decl.Kind == "widget" {
            if err := p.deleteWidget(ctx, req, decl); err != nil {
                return fmt.Errorf("widget: %w", err)
            }
        }
    }
    return nil
}
```

### Step 3: Implement reconcileX and deleteX

```go
func (p *Provider) reconcileWidget(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
    name, err := decl.Require("name")
    if err != nil {
        return err
    }
    size := decl.Field("size", "medium")

    // Read-before-write — the idempotency pattern
    existing, err := p.client.GetWidget(ctx, name)
    if isNotFound(err) {
        _, err = p.client.CreateWidget(ctx, name, size)
        if err != nil {
            return fmt.Errorf("creating widget %q: %w", name, err)
        }
        req.Logger.Info().Str("widget", name).Msg("myprovider: widget created")
        return nil
    }
    if err != nil {
        return fmt.Errorf("getting widget %q: %w", name, err)
    }

    // Drift correction
    if existing.Size != size {
        if err := p.client.UpdateWidget(ctx, name, size); err != nil {
            return fmt.Errorf("updating widget %q: %w", name, err)
        }
        req.Logger.Info().Str("widget", name).Str("size", size).Msg("myprovider: widget updated")
    }

    return nil
}

func (p *Provider) deleteWidget(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
    name := decl.Field("name", "")
    if name == "" {
        return nil
    }
    if err := p.client.DeleteWidget(ctx, name); err != nil {
        if isNotFound(err) {
            return nil // already gone — idempotent success
        }
        return fmt.Errorf("deleting widget %q: %w", name, err)
    }
    req.Logger.Info().Str("widget", name).Msg("myprovider: widget deleted")
    return nil
}
```

### Step 4: Register in `loadProviders`

```go
// internal/providers.go

func loadProviders(ctx context.Context) orktypes.ProviderRegistry {
    registry := orktypes.NewProviderRegistry()

    // ... existing providers ...

    // My provider
    p, err := myprovider.NewFromEnv()
    if err != nil {
        logger.Warn().Err(err).
            Msg("myprovider not registered — myprovider: blocks will be skipped. Set MY_API_KEY to enable.")
    } else {
        registry.Register(p)
        logger.Info().Str("provider", "myprovider").Msg("myprovider registered")
    }

    return registry
}
```

### Step 5: Add import to go.mod

```bash
go get github.com/myorg/my-external-sdk
```

### Step 6: Use in a Katalog

```yaml
providers:
  myprovider:
    - widget:
        name: "{{ .metadata.name }}-widget"
        size: "{{ default .spec.widgetSize \"medium\" }}"
        when:
          - field: spec.enableWidget
            equals: "true"
```

---

## What happens when a provider block is not registered

If the Katalog declares `myprovider:` but `myprovider` is not registered:

```
WARN  provider block skipped — no provider registered with this name.
      Import and register the provider library at startup.
      block=myprovider registered=[aws, mongodb]
```

The reconcile continues. Kubernetes resources are still managed. The operator
does not crash. This is intentional — a missing provider registration should
not take down an otherwise healthy operator.

---

## Reading credentials from Secrets

Providers must not accept raw credentials as arguments. Use `KubeReader` to
read from Kubernetes Secrets:

```yaml
# In the Katalog declaration
providers:
  myprovider:
    - widget:
        name: "{{ .metadata.name }}-widget"
        credentials:
          secretName: "{{ .metadata.name }}-myprovider-creds"
```

```go
// In the provider
func (p *Provider) reconcileWidget(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
    apiKey := ""
    if secretName := decl.Field("credentials.secretName", ""); secretName != "" {
        data, err := req.Kube.GetSecret(ctx, req.OwnerNamespace, secretName)
        if err != nil {
            return fmt.Errorf("reading credentials secret %q: %w", secretName, err)
        }
        apiKey = string(data["MY_API_KEY"])
        if apiKey == "" {
            return fmt.Errorf("secret %q missing MY_API_KEY", secretName)
        }
    }
    // use apiKey ...
}
```

The Secret must exist in the CR's namespace. Create it before applying the CR:

```bash
kubectl create secret generic my-app-myprovider-creds \
  --from-literal=MY_API_KEY=sk_live_... \
  -n default
```

---

## Testing your provider

Providers are pure Go — test them independently of Orkestra:

```go
func TestReconcileWidget(t *testing.T) {
    mockClient := &mockWidgetClient{}
    p := myprovider.New(mockClient)

    req := orktypes.ReconcileRequest{
        OwnerName:      "my-app",
        OwnerNamespace: "default",
        Declarations: []orktypes.ProviderDeclaration{
            {
                Kind: "widget",
                Fields: map[string]string{
                    "name": "my-app-widget",
                    "size": "large",
                },
            },
        },
        Kube:   &mockKubeReader{},
        Logger: zerolog.Nop(),
        Object: map[string]interface{}{},
    }

    require.NoError(t, p.Reconcile(context.Background(), req))
    assert.True(t, mockClient.createCalled)
    assert.Equal(t, "my-app-widget", mockClient.lastCreatedName)
}

// Second call — idempotency
func TestReconcileWidgetIdempotent(t *testing.T) {
    mockClient := &mockWidgetClient{existing: map[string]string{"my-app-widget": "large"}}
    p := myprovider.New(mockClient)
    // ... same request ...
    require.NoError(t, p.Reconcile(context.Background(), req))
    assert.False(t, mockClient.createCalled) // no second create
}

// Delete idempotency
func TestDeleteWidgetIdempotent(t *testing.T) {
    mockClient := &mockWidgetClient{} // nothing exists
    p := myprovider.New(mockClient)
    req := orktypes.DeleteRequest{...}
    require.NoError(t, p.Delete(context.Background(), req))
    // no error even though widget never existed
}
```

---

## Provider rules

| Rule | Why |
|---|---|
| `Reconcile` must be idempotent | Called on every resync, not just on first create |
| Return `nil` for "still provisioning" | Not an error — check again next cycle |
| Return `nil` when deleting something that doesn't exist | Idempotent delete |
| Return `error` for permanent failures only | API unreachable, invalid credentials, quota exceeded |
| Never write Kubernetes resources | Orkestra owns cluster state; providers own external state |
| Use `KubeReader` for credentials | Keeps secrets out of Katalog YAML |
| Tag external resources with `orkestra-owner` | Enables audit and ownership verification on delete |
| Log at `Info` for creates/updates/deletes | Log at `Debug` for no-ops |

---

## Current providers

| Block name | Package | Kinds |
|---|---|---|
| `aws` | `pkg/provider/aws` | `s3`, `rds`, `route53` |
| `mongodb` | `pkg/provider/mongo` | `database`, `user`, `collection` |
| `postgres` | `pkg/provider/postgres` | `database`, `role`, `extension` |
| `cache` | `pkg/provider/redis` | `acluser`, `config` |
| `mysql` | `pkg/provider/mysql` | `database`, `user` |
| `google` | `pkg/provider/google` | `gcs`, `pubsub`, `cloudsql` |
| `azure` | `pkg/provider/azure` | `blob`, `servicebus`, `sqldatabase` |

---

## Publishing to the OrkestraRegistry

Once your provider is stable, publish it as an OCI artifact:

```bash
ork provider publish \
  --name myprovider \
  --version 1.0.0 \
  --module github.com/myorg/orkestra-provider-myprovider
```

Operators that want to use your provider declare the dependency in
`pattern.yaml`:

```yaml
spec:
  providers:
    - name: myprovider
      library: oci://registry.orkestra.io/providers/myprovider:1.0.0
      required: false
```

`ork provider install` handles pulling the artifact and generating the
import scaffolding.