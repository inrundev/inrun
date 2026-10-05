# Typed

Go reconcilers compiled into your own `ork` binary. Each example is its own Go
module.

| Example | Shows |
|---------|-------|
| [hooks](hooks/) | Go hooks alongside a Katalog: YAML for what it covers, Go for the rest |
| [constructor](constructor/) | A full `domain.Reconciler` that owns the loop (a Job state machine) |
| [controller-runtime](controller-runtime/) | An unchanged controller-runtime `Reconcile` running inside Orkestra |

Every example builds the same way. First generate the type registry:

```bash
make registry
```

Then build an `ork` with the types compiled in:

```bash
make build
```

The stock `ork` fails validation for these Katalogs because it has no Go type
for the CRD; that is the reason for `make build`.
