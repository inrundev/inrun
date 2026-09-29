# Migration Guide

You have a working Kubernetes operator. The question is not whether controller-runtime works — it does. The question is what it costs.

Informers, workqueues, worker pools, leader election, status, finalizers, events, metrics, health endpoints, panic recovery — written from scratch for every CRD. The behaviour is always small. The machinery is always the same.

Orkestra removes the machinery. This guide shows the same WebApp operator expressed eight ways so you can see what you are choosing between, and when to pick each.

---

## The pack

```bash
ork init --pack from-controller-runtime
```

Each directory is a self-contained, runnable step. Work through them in order or jump to the option you need.

---

## What you will learn

- What controller-runtime costs in practice, measured against a real operator
- How to run a reconciler in any language over HTTP — no Go, no Kubernetes client
- How to go declarative — zero Go, no binary — when it fits
- How the hybrid pattern (90% declarative, 10% Go hook) works and when to use it
- How to drop all declarations and let a Go hook own every resource
- How to bring an existing `Reconcile` method into Orkestra with zero changes — two lines in a constructor are all it takes
- How `pkg/resources` simplifies the Get / Create / Patch pattern
- How `ork migrate` automates the constructor path for an existing operator file
- How a Komposer unifies all patterns into one runtime

---

## The options

| Step | Directory | Go required | What you own |
|------|-----------|-------------|--------------|
| Baseline | `00-controller-runtime-baseline` | Yes — full | Everything: informers, manager, scheme, main.go |
| Remote | `01-remote` | No | The reconciler logic — any language that speaks HTTP |
| Declarative | `02-declarative` | No | Nothing — pure YAML |
| Hybrid | `03-hybrid` | Yes — hook only | The 10% templates can't express |
| Hooks only | `04-hooks-only` | Yes — all resources | All child resource specs in Go |
| Constructor — zero change | `05-constructor-migration` | Yes — full reconciler | Reconcile unchanged; manager removed |
| Constructor — resources | `06-constructor-orkestra-resources` | Yes — full reconciler | Reconcile logic; resource ops simplified |
| ork migrate | `07-ork-migrate` | — | Automated constructor path from an existing file |
| All options | `08-all-options` | Yes — all patterns | One Komposer, five CRD kinds, one binary |

---

## Contents

| Page | What it covers |
|------|----------------|
| [The Baseline](./00-controller-runtime-baseline.md) | What controller-runtime costs — line by line |
| [Remote](./01-remote.md) | Any language over HTTP — no Go, no Kubernetes client |
| [Declarative](./02-declarative.md) | Zero Go, zero binary — pure Katalog |
| [Hybrid](./03-hybrid.md) | 90/10: declare everything Orkestra handles, write Go for the rest |
| [Hooks only](./04-hooks-only.md) | When type-safe control over every resource matters more than YAML |
| [Constructor — zero change](./05-constructor-migration.md) | Zero changes to your reconciler — two lines in a constructor |
| [Constructor — resources](./06-constructor-orkestra-resources.md) | `pkg/resources`: Get / Create / Patch → one Update call |
| [ork migrate](./07-ork-migrate.md) | Automate the constructor path for an existing operator file |
| [All options](./08-all-options.md) | All patterns in one runtime — Komposer, dependency ordering, OCI distribution |

---

## Before you begin

The pack uses a WebApp CRD. When a CR is applied, the operator creates a Deployment and a Service, then writes status. The behaviour is intentionally small so the migration story is about the machinery, not the domain.

You do not need to understand all eight options. Pick the first one that fits your situation and stop there.

**No cluster needed for simulate.** Every option has a `simulate.yaml` that runs in-memory — no cluster, no binary. Simulate before you deploy.

---

## Try it

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/02-declarative
ork simulate
ork run --dev
```

For typed operators (options 03–07), build first:

```bash
cd from-controller-runtime/03-hybrid
make registry && make build
ork simulate
ork run --dev
```

To try the automated path with your own operator:

```bash
cd from-controller-runtime/07-ork-migrate

# Follow the README
ork migrate ../00-controller-runtime-baseline/controller/webapp_controller.go -o ./output
```
