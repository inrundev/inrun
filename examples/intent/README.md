# Intent

The intent loop end to end: a caller sends a flat intent to the gateway, the
gateway turns it into a `Website` CR, the runtime reconciles it, and the caller
reads back a view of the result. The caller never writes `apiVersion`, `kind`
or `spec`.

## Without a cluster

### List the tokens

The tokens the gateway accepts, declared in `catalog.yaml`.

```bash
inrun token list
```

### Play the intent

Runs the intent through the gateway's stages (target, token, CR construction,
admission, response) and prints the CR it becomes.

```bash
inrun serve play -i intent.json --token dev
```

### Hand off to simulate

Builds the CR the same way, then reconciles it against an in-memory cluster
and checks the result against `test/simulate.yaml`.

```bash
inrun serve play -i intent.json --token dev --simulate=test/simulate.yaml
```

## On a cluster

### 1. Set the token

The value the `dev` token reads from `${INTENT_TOKEN}`.

```bash
export INTENT_TOKEN=dev-token
```

### 2. Start the runtime

Applies the CRD and reconciles.

```bash
inrun -f catalog.yaml
```

### 3. Start the gateway

In a second terminal, with the same `INTENT_TOKEN`. The runtime uses :8080 and
the console :8081.

```bash
INRUN_PORT=8082 inrun gate run -f catalog.yaml
```

### 4. Send the intent

Returns `accepted: true` and a `pollUrl`. Add `?dryRun=true` to preview
without writing.

```bash
curl -s -X POST localhost:8082/api/v1/apply \
  -H "Authorization: Bearer $INTENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d @intent.json
```

The CLI can send the same intent:

```bash
inrun serve apply -f intent.json -a http://localhost:8082 -t $INTENT_TOKEN
```

### 5. Read the view

Returns only what `serve.config.response` declares: `name`, `image`, `phase`
and `endpoint`. `phase` reads `Running` once the runtime has reconciled.

```bash
curl -s localhost:8082/api/v1/resources/website/default/hello-intent \
  -H "Authorization: Bearer $INTENT_TOKEN"
```

### 6. Clean up

Stop both processes after the script deletes the CR; it needs the runtime running to remove its finalizer.

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

### 7. E2E

Runs the whole loop in a kind cluster: sends the intent to the in-cluster gateway, reads the view back, and deletes through the gateway.

```bash
inrun e2e
```
