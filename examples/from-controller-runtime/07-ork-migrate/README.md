# 06 — ork migrate

You have an existing controller-runtime operator. Options 04 and 05 showed you where you are going. The question now is how to get there without doing the mechanical work by hand.

`ork migrate` reads your reconciler file and does the rewrite for you — signature, struct, constructor, SetupWithManager removal. What it cannot change automatically it marks with `TODO(ork migrate):` so you know exactly where to pick up. The output is a starting point, not a finished product. You resolve the TODOs, build, simulate, and the result is your operator running under Orkestra.

This option uses the `00-controller-runtime-baseline` controller as the input — the same operator you started with.

---

## Run it

```bash
ork migrate ../00-controller-runtime-baseline/controller/webapp_controller.go -o ./output
```

Compare what changed:

```bash
ork diff ../00-controller-runtime-baseline/controller/webapp_controller.go ./output/webapp_controller.go
```

---

## Work through the TODOs

```bash
grep -n "TODO(ork migrate)" ./output/webapp_controller.go
```

Resolve each one — add the Orkestra imports, replace the sub-method client calls, update the status patch — then build and simulate:

```bash
cd output
make registry
make build
ork simulate
```

When the simulation passes, the result is equivalent to `04-constructor-migration`.

---

## Next

[07 — All options](../07-all-options/README.md) — run all five operator patterns together in a single Orkestra runtime.

---

## See also

- [04 — constructor: lift and change](../04-constructor-migration/) — what the output looks like after the TODOs are resolved
- [05 — constructor: Orkestra resources](../05-constructor-orkestra-resources/) — the next step once it builds
- [CLI reference: ork migrate](https://orkestra.sh/docs/reference/cli/migrate/)
