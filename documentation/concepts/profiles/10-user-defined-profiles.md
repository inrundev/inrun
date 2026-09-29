# User-Defined Profiles

Built-in Orkestra profiles cover the common cases. User-defined profiles let your team define the cases specific to you — your network topology, your compliance requirements, your capacity tiers — and name them so that intent travels through every Katalog and Motif that imports them.

---

## Declaring profiles

Profiles are declared at the root of a Katalog or Motif alongside `spec:` and `security:`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: platform-operator

profiles:
  reconcile:
    - name: api-service
      workers: 4
      resync: 30s
      queue:
        maxDepth: 200
  networkPolicies:
    - name: allow-monitoring
      description: Allow ingress from the platform monitoring namespace
      ingress:
        - from:
            - namespaceSelector:
                team: platform
      policyTypes: [Ingress]

  resourceQuotas:
    - name: team-medium
      description: Standard allocation for a medium-sized team namespace
      hard:
        pods: "30"
        cpu: "6"
        memory: "12Gi"
        requests.cpu: "3"
        requests.memory: "6Gi"

spec:
  crds:
    namespace:
      ...
      operatorBox:
        reconcile:
          profile: api-service
          onCreate:
            networkPolicies:
              - name: "{{ .metadata.name }}-monitoring"
                profile: allow-monitoring
            resourceQuotas:
              - name: "{{ .metadata.name }}-quota"
                profile: team-medium
```

---

## Supported profile classes

| Class | YAML key | Expands into | Referenced from |
|-------|----------|--------------|-----------------|
| NetworkPolicy | `profiles.networkPolicies` | ingress/egress rules, policyTypes | `onCreate.networkPolicies[].profile` |
| ResourceQuota | `profiles.resourceQuotas` | hard limits map | `onCreate.resourceQuotas[].profile` |
| LimitRange | `profiles.limitRanges` | limit items | `onCreate.limitRanges[].profile` |
| HPA | `profiles.hpa` | minReplicas, maxReplicas, CPU target, behavior | `onCreate.hpa[].behavior.profile` |
| PDB | `profiles.pdb` | minAvailable or maxUnavailable | `onCreate.pdb[].behavior.profile` |
| Rolling Update | `profiles.rollingUpdate` | maxSurge, maxUnavailable | `onCreate.deployments[].rollingUpdate.profile` |
| Reconciler | `profiles.reconciler` | workers, resync, queue.maxDepth | `operatorBox.reconcile.profile` |
| Resources | `profiles.resources` | requests and limits per container | `containers[].resources.profile` |
| Probes | `profiles.probes` | initialDelaySeconds, periodSeconds, failureThreshold, successThreshold, timeoutSeconds | `containers[].probes[].profile` |
| Container Security | `profiles.containerSecurity` | allowPrivilegeEscalation, readOnlyRootFilesystem, runAsNonRoot, capabilities | `containers[].securityContext.profile` |
| Pod Security | `profiles.podSecurity` | runAsNonRoot, runAsUser, runAsGroup, fsGroup | `spec.securityContext.profile` |

> **Not yet supported:** `operatorBox.autoscaler` — the operator autoscaler does not support user-defined profiles. Configure it inline.

The reconciler class is different from the others: it tunes how the CRD's own reconciler runs, not what child resources are created. It is set once per CRD entry at the `operatorBox.reconcile` level rather than on individual child resource entries.

### Reconciler profiles

A reconciler profile sets the tuning for a CRD's reconcile loop — workers, resync interval, and queue depth. Declare profiles in `profiles.reconciler` and reference one with `operatorBox.reconcile.profile`:

```yaml
profiles:
  reconcile:
    - name: api-service
      description: Balanced for a standard web service operator
      workers: 4
      resync: 30s
      queue:
        maxDepth: 200

    - name: batch-worker
      description: Low churn, background processing
      workers: 2
      resync: 5m
      queue:
        maxDepth: 500
spec:
  crds:
    orders:
      operatorBox:
        reconcile:
          profile: api-service
          # inline fields override the profile — add here to tune per-CRD
    archive:
      operatorBox:
        reconcile:
          profile: batch-worker
```

Inline fields always win over the profile. To use a profile as the baseline and tune one field:

```yaml
operatorBox:
  reconcile:
    profile: api-service
    workers: 8    # override — profile's workers (4) is ignored; resync and maxDepth come from the profile
```

Orkestra also ships three built-in reconciler profiles that require no declaration:

| Profile | workers | resync | queue.maxDepth |
|---------|---------|--------|----------------|
| `high-throughput` | 10 | 5m | 1000 |
| `conservative` | 2 | 1m | 100 |
| `development` | 1 | 30s | 50 |

### Resources profiles

A resources profile sets container CPU and memory requests and limits. Declare in `profiles.resources` and reference with `containers[].resources.profile`:

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

    - name: batch-processor
      requests:
        cpu: "1"
        memory: "512Mi"
      limits:
        cpu: "4"
        memory: "2Gi"
```

Orkestra ships built-in resource profiles (`tiny`, `small`, `medium`, `large`, `burst`, `steady`, `compute-heavy`, `memory-heavy`) that require no declaration.

### Probes profiles

A probes profile sets timing parameters shared across liveness, readiness, and startup probes:

```yaml
profiles:
  probes:
    - name: slow-boot
      description: For services with long JVM or Python startup times
      initialDelaySeconds: 60
      periodSeconds: 20
      failureThreshold: 5
      successThreshold: 1
      timeoutSeconds: 10
```

Orkestra ships built-in probe profiles (`fast`, `standard`, `patient`, `slow-start`) that require no declaration.

### Container security profiles

A container security profile sets securityContext fields on individual containers:

```yaml
profiles:
  containerSecurity:
    - name: strict-readonly
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      runAsNonRoot: true
      capabilities:
        drop: [ALL]
```

Orkestra ships built-in container security profiles (`baseline`, `restricted`, `hardened`) that require no declaration.

### Pod security profiles

A pod security profile sets securityContext fields at the pod level:

```yaml
profiles:
  podSecurity:
    - name: ci-runner
      runAsNonRoot: true
      runAsUser: 2000
      runAsGroup: 2000
      fsGroup: 2000
```

Orkestra ships built-in pod security profiles (`baseline`, `restricted`, `hardened`) that require no declaration.

---

## Template expressions in profile fields

Profile field values support template expressions. They are resolved at reconcile time against the live CR:

```yaml
profiles:
  resourceQuotas:
    - name: cr-sized
      hard:
        pods: "{{ .spec.maxPods }}"
        cpu: "{{ .spec.cpuLimit }}"
        memory: "{{ .spec.memLimit }}"

  hpa:
    - name: cr-scaled
      minReplicas: "{{ .spec.minReplicas | default \"2\" }}"
      maxReplicas: "{{ .spec.maxReplicas }}"
      targetCPUUtilizationPercentage: "70"
```

At `ork validate` time, fields containing `{{` are skipped — they cannot be validated statically. At reconcile time, the expression is expanded before the profile is applied.

---

## Validation

`ork validate` enforces three rules on the `profiles:` block:

1. **Non-empty name** — every profile entry must declare a `name`.
2. **Unique within class** — two `networkPolicies` entries with the same name is an error. Two entries with the same name in different classes (`resourceQuotas` and `hpa`) is fine — class is the scope boundary.
3. **Shadowing built-ins** — allowed but warned. If you declare a `networkPolicies` profile named `deny-all`, your version is used instead of Orkestra's built-in. A warning is printed at validate time so the shadowing is explicit.

---

## Resolution order

When a resource declares `profile: some-name`, Orkestra resolves it in this order:

1. User-defined profiles in the Katalog `profiles:` block
2. User-defined profiles merged from Motifs imported at `spec.imports`
3. Built-in Orkestra profiles

The first match wins. Built-ins are only consulted when the name is not found in any user registry.

---

## Profiles in Motifs

A Motif can declare its own `profiles:` block. To make those profiles available across the Katalog, import the Motif at `spec.imports` — profiles are only merged at this level. Importing the same Motif at `spec.crds[name].imports` does not merge its profiles (only resources, status, and admission are consumed there).

```yaml
# tenant-isolation.motif.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: tenant-isolation
  version: v0.2.0

profiles:
  networkPolicies:
    - name: allow-monitoring
      ingress:
        - from:
            - namespaceSelector:
                team: platform
      policyTypes: [Ingress]
    - name: allow-internal
      ingress:
        - from:
            - namespaceSelector:
                scope: internal
      policyTypes: [Ingress]

resources:
  networkPolicies:
    - name: "{{ .metadata.name }}-deny-all"
      profile: deny-all
    - name: "{{ .metadata.name }}-monitoring"
      profile: allow-monitoring
```

Import at `spec.imports` to merge profiles into the Katalog-wide registry:

```yaml
spec:
  imports:
    - motif: ./motifs/tenant-isolation.yaml   # profiles merged here

  crds:
    namespace:
      operatorBox:
        reconcile:
          onCreate:
            networkPolicies:
              - name: "{{ .metadata.name }}-monitoring"
                profile: allow-monitoring   # ✓ available to all CRDs
    service:
      operatorBox:
        reconcile:
          onCreate:
            networkPolicies:
              - name: "{{ .metadata.name }}-monitoring"
                profile: allow-monitoring   # ✓ same profile, no re-import needed
```

### Conflict detection

If the same profile name appears in the same class in both the Katalog and an imported Motif — or in two imported Motifs — it is a **hard error** at load time:

```text
profile conflict: networkPolicies "allow-monitoring" defined in both motif "tenant-isolation" and the katalog
```

The same name in different classes is not a conflict — `resourceQuotas.medium` and `hpa.medium` are independent.

---

## ork validate output

Pass `--profiles` when validating a Katalog to print the merged profile registry — your inline declarations plus anything imported from Motifs:

```bash
ork validate --profiles
```

```text
Profiles (9)
  resources           team-api
  containerSecurity   team-baseline
  podSecurity         team-nonroot
  hpa                 team-conservative
  hpa                 team-responsive
  networkPolicies     team-deny-all
  networkPolicies     team-allow-internal
  networkPolicies     team-allow-dns
  probes              team-standard
```

When validating a Motif directly, `ork validate` shows profiles alongside resources in the summary:

```text
● tenant-isolation
  Reusable network isolation motif
  version  : v0.2.0
  inputs   : 2
  resources: networkPolicies(1)
  profiles : networkPolicies(2)
```

---

→ Back: [09 — NetworkPolicy Profile](09-networkpolicy-profile.md) | [Profiles index](index.md)
