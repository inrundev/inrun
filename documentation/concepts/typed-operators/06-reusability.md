# Reusability

A typed operator in Orkestra is reusable at two independent levels: **API** and **configuration**.

---

## API reusability — one binary, many CR types

Whatever you declare under `apiTypes` is what Orkestra delivers to your hook or constructor. Multiple teams can each ship a Katalog with their own CRD name and Kind — all in the same cluster, all backed by the same binary.

This is the actual reusability: the Kind is different in each Katalog. Team A ships a `PostgresCluster` operator, team B ships a `ManagedDatabase` operator — both running in the same cluster, both backed by the same hooks binary.

```yaml
# team-a/katalog.yaml
crds:
  postgres-cluster:
    apiTypes:
      group: db.team-a.io
      version: v1
      kind: PostgresCluster
      plural: postgresclusters
      object: PostgresCluster
      objectList: PostgresClusterList
      location: github.com/myorg/db-hooks/api/v1
    operatorBox:
      reconcile:
        hooks:
          location: github.com/myorg/db-hooks/hooks
          function: DatabaseHooks
# team-b/katalog.yaml — same cluster, different CRD and Kind, same binary
crds:
  managed-database:
    apiTypes:
      group: db.team-b.io
      version: v1alpha1
      kind: ManagedDatabase
      plural: manageddatabases
      object: ManagedDatabase
      objectList: ManagedDatabaseList
      location: github.com/myorg/db-hooks/api/v1alpha1
    operatorBox:
      reconcile:
        hooks:
          location: github.com/myorg/db-hooks/hooks
          function: DatabaseHooks   # same binary
```

The Go types at `location` are the author's — they can differ between Katalogs. Orkestra decodes the API server response into whichever type the hook declared. The hook is unaware of which Katalog loaded it or which Kind triggered the reconcile.

---

## Configuration reusability — one binary, many deployments

`args:` lets a single hook or constructor handle different environments, tiers, or tenants — all configured from the Katalog without code changes.

Args have two modes: **static** values (integers, booleans, and plain strings without `{{ }}`) that are fixed at startup, and **dynamic** values (strings with `{{ }}`) that are evaluated per-CR at reconcile time using the full note FuncMap. Both modes compose freely in the same `args:` block — including inside nested maps, which are recursed into so any string at any depth can be a template.

A platform constructor that serves three tiers from one binary:

```yaml
crds:
  app-starter:
    operatorBox:
      reconcile:
        constructor:
          location: github.com/myorg/platform/constructor
          function: PlatformConstructor
          args:
            tier: starter
            regions:
              - name: eu-west-1
                primary: true
            resources:
              requests: { cpu: "100m", memory: "128Mi" }
              limits:   { cpu: "500m", memory: "512Mi" }
            features:
              hpa: false
              pdb: false
  app-standard:
    operatorBox:
      reconcile:
        constructor:
          location: github.com/myorg/platform/constructor
          function: PlatformConstructor   # same binary
          args:
            tier: standard
            regions:
              - name: eu-west-1
                primary: true
            resources:
              requests: { cpu: "200m", memory: "256Mi" }
              limits:   { cpu: "1",    memory: "1Gi"   }
            features:
              hpa: true
              hpaMin: 2
              hpaMax: 10
              pdb: true
  app-enterprise:
    operatorBox:
      reconcile:
        constructor:
          location: github.com/myorg/platform/constructor
          function: PlatformConstructor   # same binary
          args:
            tier: enterprise
            regions:
              - name: eu-west-1
                primary: true
              - name: us-east-1
                primary: false
              - name: ap-southeast-1
                primary: false
            resources:
              requests: { cpu: "500m", memory: "512Mi" }
              limits:   { cpu: "4",    memory: "4Gi"   }
            features:
              hpa: true
              hpaMin: 3
              hpaMax: 50
              pdb: true
              multiRegion: true
            compliance:
              soc2: true
              auditRetentionDays: 90
```

Three CRDs, one constructor, all differences in YAML. The constructor reads the args via `kube.Args()` and branches on them — no environment variables, no hardcoded tiers, no duplicate binaries.

The `tier` and feature flags are static. You can also mix in dynamic args that pull from each individual CR at reconcile time:

```yaml
args:
  tier: enterprise
  # static — same for every CR
  features:
    hpa: true
    hpaMin: 3
  # dynamic — resolved from each CR individually
  tenantId: "{{ .spec.tenantId }}"
  region:   '{{ default "eu-west-1" .spec.region }}'
```

A single Katalog declaration serves all CRs of that type. Each reconcile resolves `tenantId` and `region` from the CR being reconciled — no per-tenant Katalog needed.

```go
func PlatformConstructor(
    kube kubeclient.KubeClient,
    informer cache.SharedIndexInformer,
    ev event.Recorder,
) domain.Reconciler {
    var cfg PlatformArgs
    if err := kube.Args().BindArgs(&cfg); err != nil {
        // surfaced by ork validate before the operator starts
        panic(fmt.Sprintf("invalid platform constructor args: %v", err))
    }
    return &PlatformReconciler{kube: kube, informer: informer, event: ev, cfg: cfg}
}
```

`cfg` carries the full tier configuration — resources, features, compliance — resolved once at startup from the Katalog declaration.

---

## Composing both dimensions

Both reusability axes are independent and composable. The same binary can target different API types in different Katalogs AND receive different args in each:

```yaml
# eu-production/katalog.yaml
crds:
  database:
    apiTypes:
      kind: Database
      location: github.com/myorg/db-operator/api/v1
    operatorBox:
      reconcile:
        hooks:
          function: DatabaseHooks
          args:
            region: eu-west-1
            backup: { enabled: true, scheduleHour: 2 }
# us-production/katalog.yaml — different region and schedule
crds:
  database:
    apiTypes:
      kind: Database
      location: github.com/myorg/db-operator/api/v1
    operatorBox:
      reconcile:
        hooks:
          function: DatabaseHooks   # same binary
          args:
            region: us-east-1
            backup: { enabled: true, scheduleHour: 4 }
```

---

## Komposer overrides

A Komposer can override specific args without redeclaring the full map. Only declared keys are overridden; siblings are preserved.

```yaml
# komposer.yaml — enterprise override for SOC2 audit retention
spec:
  crds:
    app-enterprise:
      operatorBox:
        reconcile:
          constructor:
            args:
              compliance:
                auditRetentionDays: 365   # overrides the Katalog value
                                          # soc2: true is preserved from Katalog
```

This follows the same server-side apply semantics as all other Orkestra merge operations — declare what changes, leave the rest untouched.

---

## What you maintain in one place

| Layer | Where it lives |
|-------|----------------|
| Reconcile logic | Your Go binary |
| API type shape | `apiTypes.location` in the Katalog |
| Environment/tier/tenant configuration | `args:` in the Katalog |
| Per-deployment overrides | Komposer |

The Go binary knows nothing about which tier it serves, which region it runs in, or which team deployed it. All of that is in YAML, where operators see and version it alongside their CRD definitions.
