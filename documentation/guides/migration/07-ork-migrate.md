# ork migrate

Options 04 and 05 showed you what your operator looks like under Orkestra. The signature change, the struct rewrite, the constructor, the SetupWithManager removal — all of it follows the same mechanical pattern regardless of the operator. `ork migrate` does the mechanical part for you.

Point it at your reconciler file:

```bash
ork migrate ./controller/webapp_controller.go -o ./output
```

It rewrites the signature, replaces the struct, generates the constructor, removes `SetupWithManager`, and emits `katalog.yaml`, `simulate.yaml`, `e2e.yaml`, `go.mod`, `Makefile`, and `Dockerfile`. What it cannot change safely it marks with `TODO(ork migrate):` so you know exactly where to pick up.

---

## After running

```bash
grep -n "TODO(ork migrate)" ./output/webapp_controller.go
```

The TODOs cover what the tool cannot infer from syntax alone:
- Add Orkestra imports (`domain`, `kubeclient`, `event`, `cache`)
- Replace sub-method client calls (`r.Get`, `r.Create`, `r.Patch`) with `r.kube.*`
- Replace `r.Status().Update()` with `r.kube.PatchStatus()`
- Set `group` and `plural` in `katalog.yaml` — `kind`, `version`, `location`, `alias`, `object`, `objectList`, owned `managedResources:`, and `watch:` entries are auto-detected from `SetupWithManager`
- Fill in simulate assertions and e2e setup

Once resolved, build and simulate:

```bash
make registry && make build
./ork simulate
```

---

## Flags

```bash
ork migrate <file>                     # prompts before replacing in place
ork migrate <file> -o <dir>           # write to directory — original unchanged
ork migrate <file> --module <module>  # set Go module path in go.mod
ork migrate <file> --name <operator>  # set operator name in kebab-case
```

---

## Try it

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/07-ork-migrate
# Follow steps in README
```

---

## Full reference

→ [CLI reference: ork migrate](../reference/cli/migrate.md)
