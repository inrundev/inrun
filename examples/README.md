# Examples

The intent loop, from the reconciler side and from the caller side. Each pack
is embedded in the CLI:

```bash
ork init my-operator --pack declarative    # or remote, typed, intent
ork init --list
```

| Pack | Where | Shows |
|------|-------|-------|
| `declarative` | [reconcilers/declarative](reconcilers/declarative/) | Operators declared in a Katalog, no code. Start here. |
| `remote` | [reconcilers/remote](reconcilers/remote/) | A reconciler in any language, called over HTTP |
| `typed` | [reconcilers/typed](reconcilers/typed/) | Go hooks, a constructor, and a controller-runtime drop-in |
| `intent` | [intent](intent/) | Intent in through the gateway, view out |

Every example has the same layout:

```
katalog.yaml      the operator; ork run, validate and generate read it
manifests/        what is applied to the cluster: the CRD, a CR, setup objects
test/             simulate.yaml, e2e.yaml and their values; found by ork simulate and ork e2e
cleanup.sh
```
