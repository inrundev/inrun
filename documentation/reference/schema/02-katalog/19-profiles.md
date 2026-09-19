# profiles

Named profile definitions declared at the Katalog or Motif level. Profiles are resolved by name at reconcile time — user-defined profiles take precedence over built-ins with the same name.

```yaml
profiles:
  resources:
    - name: api-worker
      requests:
        cpu: "500m"
        memory: "256Mi"
      limits:
        cpu: "2"
        memory: "1Gi"

  probes:
    - name: slow-boot
      initialDelaySeconds: 60
      periodSeconds: 20
      failureThreshold: 5

  networkPolicies:
    - name: allow-monitoring
      ingress:
        - from:
            - namespaceSelector:
                team: platform
      policyTypes: [Ingress]
```

## `profiles.include`

Pull profile definitions from an external file to keep the Katalog compact. The file contains the same structure as the inline `profiles:` block — any subset of the 11 profile sub-lists.

```yaml
profiles:
  include: ./profiles/shared.yaml   # relative to the katalog file
  resources:
    - name: local-dev               # appended after included definitions
      requests:
        cpu: "100m"
        memory: "64Mi"
```

`profiles/shared.yaml`:

```yaml
profiles:
  resources:
    - name: api-worker
      requests:
        cpu: "500m"
        memory: "256Mi"
      limits:
        cpu: "2"
        memory: "1Gi"
  probes:
    - name: slow-boot
      initialDelaySeconds: 60
      periodSeconds: 20
      failureThreshold: 5
```

Included definitions come first per sub-list. Inline entries append after. If the same name appears in the same class in both, it is a **hard error** — no silent override. The `include:` path resolves relative to the katalog file's directory. The field is cleared from the runtime bundle after expansion.

## Profile classes

| Key | Expands into | Referenced from |
|-----|--------------|-----------------|
| `profiles.networkPolicies` | ingress/egress rules, policyTypes | `onCreate.networkPolicies[].profile` |
| `profiles.resourceQuotas` | hard limits map | `onCreate.resourceQuotas[].profile` |
| `profiles.limitRanges` | limit items | `onCreate.limitRanges[].profile` |
| `profiles.hpa` | minReplicas, maxReplicas, CPU target, behavior | `onCreate.hpa[].behavior.profile` |
| `profiles.pdb` | minAvailable or maxUnavailable | `onCreate.pdb[].behavior.profile` |
| `profiles.rollingUpdate` | maxSurge, maxUnavailable | `onCreate.deployments[].rollingUpdate.profile` |
| `profiles.reconciler` | workers, resync, queue.maxDepth | `operatorBox.reconcile.profile` |
| `profiles.resources` | requests and limits per container | `containers[].resources.profile` |
| `profiles.probes` | probe timing parameters | `containers[].probes[].profile` |
| `profiles.containerSecurity` | allowPrivilegeEscalation, readOnlyRootFilesystem, runAsNonRoot, capabilities | `containers[].securityContext.profile` |
| `profiles.podSecurity` | runAsNonRoot, runAsUser, runAsGroup, fsGroup | `spec.securityContext.profile` |

## Distributing via Motifs

A Motif can declare a `profiles:` block. Import the Motif at `spec.imports` to merge its profiles into the Katalog-wide registry — they become available to every CRD entry.

```yaml
# motif.yaml
profiles:
  networkPolicies:
    - name: allow-monitoring
      ingress:
        - from:
            - namespaceSelector:
                team: platform
      policyTypes: [Ingress]
```

```yaml
# katalog.yaml
spec:
  imports:
    - motif: ./motifs/tenant-isolation.yaml
```

A Motif can also use `profiles.include:` — the path resolves relative to the Motif file.

## Conflict rules

| Situation | Result |
|-----------|--------|
| Same name, same class, two `spec.imports` Motifs | Hard error at startup |
| Same name, same class, inline and a Motif import | Hard error at startup |
| Same name, different classes | Fine — class is the scope boundary |
| Name matches a built-in | Warning logged; user definition wins |

## Where to go next

- [User-Defined Profiles concept](../../../concepts/profiles/10-user-defined-profiles.md) — full sub-class reference, template expressions, validation, resolution order
- [Motif schema](../01-motif/index.md) — `profiles:` on a Motif
