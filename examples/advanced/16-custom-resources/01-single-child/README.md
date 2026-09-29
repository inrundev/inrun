# 01 — Single Child CR

A `Workspace` CRD automatically provisions a `SecretVault` child CR the moment it is created. The SecretVault operator then spins up the secrets API server Deployment. Delete the Workspace and everything cascades away cleanly via owner references.

---

## What This Shows

The absolute basics of `onCreate.custom`:

1. Parent CR (`Workspace`) is applied.
2. Orkestra resolves template expressions and creates the `SecretVault` child CR.
3. The SecretVault operator picks up the new CR and creates a Deployment + Service.
4. Deleting the Workspace cascades to the SecretVault (and its Deployment) via owner references.

---

## New Concepts Introduced

### `onCreate.custom`

Declares child custom resources to create when the parent CR is first reconciled. Each entry is a full CR manifest with Go-template expressions resolved against the parent.

```yaml
operatorBox:
  reconcile:
    onCreate:
      custom:
        - apiVersion: platform.example.io/v1alpha1
          kind: SecretVault
          metadata:
            name: "{{ .metadata.name }}-vault"
            namespace: "{{ .metadata.namespace }}"
            namespaced: true
          spec:
            workspaceName: "{{ .metadata.name }}"
            encryption: "{{ .spec.encryption }}"
            maxSecrets: 100
          hasStatus: false
```

Key fields:

| Field | Purpose |
|---|---|
| `namespaced: true` | Creates the child in the same namespace as the parent |
| `hasStatus: false` | Tells Orkestra not to read child status back into parent — saves an API call |
| Template expressions | `{{ .metadata.name }}`, `{{ .spec.encryption }}` etc. are resolved from the parent CR |

### Owner References (automatic)

Orkestra automatically sets an owner reference on every child CR pointing back to the parent. This means:

```bash
kubectl delete workspace dev-team
# SecretVault dev-team-vault is garbage-collected automatically
# Deployment/Service created by SecretVault operator also cascade away
```

No manual cleanup of child resources is needed.

### Two Operators, One Katalog

Both `workspace` and `secretvault` are declared in the same `katalog.yaml`. Orkestra runs a separate controller goroutine for each. The SecretVault operator has no idea it was created by Workspace — it just sees a SecretVault CR and creates its own Deployment.

---

## Prerequisites

- `kubectl` configured to a running cluster (Kind works)
- Ork CLI:
  ```bash
  curl get.orkestra.sh | bash
  ```

---

## Run the Example

### 1. Apply the CRDs

```bash
kubectl apply -f crd-workspace.yaml
kubectl apply -f crd-secretvault.yaml
```

### 2. Start the runtime

```bash
ork run 
```

Leave this running in a terminal. You should see startup logs confirming both controllers are ready.

### 3. Apply the Workspace CR

```bash
kubectl apply -f cr.yaml
```

### 4. Observe the chain

Within a few seconds:

```bash
# Parent Workspace is Ready
kubectl get workspaces -n default

# SecretVault child was created automatically
kubectl get secretvaults -n default

# SecretVault operator created a Deployment
kubectl get deployments -n default
```

Expected output:

```
NAME       TEAM           ENCRYPTION   PHASE   AGE
dev-team   platform-eng   AES256       Ready   10s

NAME             ENCRYPTION   MAXSECRETS   PHASE     AGE
dev-team-vault   AES256       100          Running   8s
```

### 5. Verify owner reference

The SecretVault has an owner reference pointing to the Workspace:

```bash
kubectl get secretvault dev-team-vault -n default -o jsonpath='{.metadata.ownerReferences}' | jq .
```

### 6. Delete the Workspace and watch cascade

```bash
kubectl delete workspace dev-team -n default
```

Then verify everything is gone:

```bash
kubectl get workspaces,secretvaults,deployments -n default
# Expected: No resources found
```

---

## What to Observe

- The `SecretVault` is created with the resolved name `dev-team-vault` (the template `{{ .metadata.name }}-vault` where parent is `dev-team`).
- The Workspace status reports `vaultRef: dev-team-vault`.
- The SecretVault has `ownerReferences` pointing to the Workspace UID.
- Deleting the Workspace removes the entire chain without any manual steps.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
