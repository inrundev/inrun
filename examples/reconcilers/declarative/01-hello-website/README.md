# 01-hello-website

One `Website` CR becomes a Deployment and a Service, owned by the CR, with
`phase` and `endpoint` written to its status.

### 1. Validate

```bash
ork validate
```

### 2. Run

```bash
ork run
```

### 3. Check the status

In a second terminal.

```bash
kubectl get website hello-website -o jsonpath='{.status}'; echo
```

### 4. Delete the Deployment

It comes back on the next reconcile.

```bash
kubectl delete deployment hello-website
```

### 5. Scale through the CR

```bash
kubectl patch website hello-website --type=merge -p '{"spec":{"replicas":2}}'
```

### 6. Clean up

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

### 7. E2E

Runs the same checks in a kind cluster.

```bash
ork e2e
```
