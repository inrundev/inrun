# Declarative

Operators declared entirely in a Katalog: no code. Work through them in order.

| Example | Shows |
|---------|-------|
| [01-hello-website](01-hello-website/) | One CR → a Deployment and a Service, with status |
| [02-with-serviceaccount](02-with-serviceaccount/) | A ServiceAccount wired into the Deployment; status read from live children |
| [03-secret-copy](03-secret-copy/) | A built-in kind: copy a Secret into other namespaces and keep it in sync |
| [04-configmap-copy](04-configmap-copy/) | The same pattern for a ConfigMap |

### Run one

```bash
cd 01-hello-website
```

```bash
ork run
```

### Simulate all of them

No cluster needed.

```bash
ork simulate
```

### End to end

Every example in one kind cluster.

```bash
ork e2e
```
