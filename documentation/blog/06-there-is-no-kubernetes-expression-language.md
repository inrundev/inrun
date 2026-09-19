# There Is No Kubernetes Expression Language

*So I built one.*

---

I was building the Katalog and kept reaching for Go. A function here. A helper there. Every time I needed to reason about something — is this deployment healthy, are all replicas ready, is this pod crashing — I wrote Go.

At some point I stopped and asked a question.

*Does the Kubernetes schema actually change?*

Deployments, Services, Conditions, Pods — the shape is fixed. The API is stable. Nobody invents a new kind of Deployment condition. What changes is the reasoning people want to do on top of it. And if the schema is fixed, the knowledge about it can be encoded once.

Then I asked a second question.

*What do people actually want to reason about when they work with operators?*

The interesting part of operator logic is always a condition — should I do this, given what I know. That's if/else. Everything else is setup. And on top of that: data transformation, time, safe access to deeply nested fields. And ultimately: is this deployment healthy.

If those questions have fixed answers, and the schema they interrogate is stable, then the knowledge belongs in a library. Written once. Available everywhere. Not rewritten in every controller until the project ends and it disappears.

That was when Go templates stopped being a rendering engine and became the right place to put it.

---

Every team that operates Kubernetes eventually writes the same functions.

`hasCrashingPod`. `rolloutComplete`. `allReplicasReady`. `isTerminating`. The names vary slightly. The logic is identical. Someone writes it in their controller, it lives in that codebase, and when the project ends it disappears. The next team starts from scratch.

This bothered me for a long time before I had a name for it.

---

## The tools we borrowed

The Kubernetes ecosystem has reached for expression languages several times.

**Helm** borrowed Go templates. It worked — for static resource rendering. But Go templates don't know what a Kubernetes condition is. They don't know what `CrashLoopBackOff` means. They can interpolate values, but they can't reason about a pod's health state. The moment you need to say "scale up only if there are no crashing pods," you're outside what Helm can express.

**CEL** — the Common Expression Language — is used in admission webhooks and validation rules. It is genuinely powerful. Strongly typed, fast, sandboxed. But it knows nothing about Kubernetes. It knows about strings and integers and lists. If you want to check whether a Deployment's rollout is stalled, you write the logic yourself, every time, in every place you need it.

**OPA Rego** is a full policy language. Expressive enough to encode almost any rule. But it is a language you have to learn before you can say "does this pod have a restart count above threshold." And what you write in Rego stays in Rego. It does not compose with your reconciler. It does not run in your E2E tests. It does not appear in your CLI.

**Kyverno** policies. **ValidatingAdmissionPolicies**. **Custom controllers** with embedded logic. Every approach solves a local problem and leaves the knowledge behind when the context changes.

The pattern is always the same: borrow a general-purpose tool, teach it Kubernetes by hand, use the result in one place, repeat from scratch everywhere else.

---

## The more I looked, the more Go I had to write

When I started building the [Katalog](../getting-started/03-writing-your-first-katalog.md), I needed it to know about Kubernetes. Not just to template values into manifests — Helm already did that. I needed the Katalog to *reason* about Kubernetes objects at runtime. Is this deployment healthy? Are all replicas ready? Is this pod in a crash loop?

Every time I added that kind of knowledge to the Katalog, I reached for Go. A function here. A helper there. Before long I had a growing pile of Go code — none of it the interesting part, all of it the same kind of structural navigation of Kubernetes objects that I had seen written and rewritten across every controller I had ever read.

I kept thinking: this shouldn't live in Go. The person writing a Katalog shouldn't need to know Go to ask whether a pod is crashing. The knowledge should be available to the YAML author, at declaration time, without a compile step.

Then it dawned on me. Go ships a templating engine — `text/template` — and it is extensible. You give it a `FuncMap` and every function in that map becomes available to every expression the engine evaluates. No new syntax. No new parser. Just functions — and Go functions can do anything, including understanding Kubernetes objects.

Helm had used the same mechanism to add `toYaml` and `indent`. Those functions don't know what a condition is. They don't know what `CrashLoopBackOff` means. But they proved the point — the engine was extensible, the community had embraced it, and the only thing missing was functions that actually knew Kubernetes.

I started adding functions that knew Kubernetes.

Not new syntax. Not a new parser. Not a new language to learn. Just functions — pure, stateless, composable — that understood what a Kubernetes object was and could answer questions about it.

```text
hasCrashingPod(deployment) → bool
allReplicasReady(deployment) → bool
rolloutComplete(deployment) → bool
isTerminating(obj) → bool
conditionReason(obj, type) → string
isSynced(obj) → bool
```

These functions take a Kubernetes object — the raw `map[string]interface{}` the dynamic client returns — and return a typed answer. They handle nil. They handle missing fields. They handle the subtle differences between how different Kubernetes types represent the same concept. They encode the knowledge that used to live in every team's private controller and disappear when the project ended.

---

## The moment it clicked

The functions were registered in the template engine. What I hadn't fully appreciated was where the template engine ran.

It ran in the reconciler — evaluating `when:` conditions before creating resources.

It ran in the admission webhook — evaluating validation rules before accepting a CR.

It ran in the E2E test runner — asserting on field values in `kubectl get` output.

It ran in the CLI — when you search for notes or preview what a template expression would return.

Same function. Same registration. Same behavior. Written once.

When I wrote `{{ timeSince (creationTimestamp .children.deployment) }}` and it worked identically in an E2E assertion and a live admission webhook and a local `ork run` session — that was the moment the idea became a name.

---

## KEL — Kubernetes Expression Language

Not a new language. A library.

The distinction matters. A language has a parser, a grammar, a runtime. It requires tooling, documentation, a learning curve. A library has functions. You call them. You compose them. You extend them by adding more functions.

KEL is a composable vocabulary for expressing Kubernetes operational knowledge inside any template expression. The host is Go's template engine. The vocabulary is the function catalog. The composition is just pipes:

```text
'{{ timeSince (creationTimestamp .children.deployment) }}'
```

That expression is three KEL primitives: `creationTimestamp` reads a field, `timeSince` measures elapsed seconds, and Go's template pipeline connects them. No new syntax. No new runtime. The result is available wherever a template expression is evaluated.

The same is true for any combination:

```text
'{{ getAnnotation . "autoscale/min-replicas" | toInt | default 2 }}'
'{{ labelMatches .children.deployment "env" "prod" }}'
'{{ hasStatus .children.service "loadBalancer" }}'
'{{ promAboveThreshold .external.queueDepth 10000 }}'
```

Each expression is a policy. Each policy is a composition of primitives. Each primitive is written once and available everywhere.

---

## Why CEL doesn't solve this

CEL is powerful. I want to be honest about that. For admission policies that validate field values against schema constraints, CEL is the right tool. It is fast, sandboxed, and standardized.

But CEL evaluates against a schema. It does not evaluate against a running system. It cannot ask "are all replicas of this deployment ready right now?" because that question requires navigating `status.conditions`, checking specific type strings, comparing integer fields — knowledge that is encoded in the Kubernetes API spec, not in any expression language.

KEL can ask that question because `allReplicasReady` encodes the answer. It knows the shape of a Deployment's status. It knows what `Available` means in a condition. It returns a boolean you can compose with anything else.

The other difference is scope. CEL runs where it is configured to run. KEL runs wherever the template engine runs — which in Orkestra is everywhere. The same expression that gates a Deployment in a reconciler gates a CR at admission time and asserts a field value in an E2E test. The knowledge doesn't live in one place. It lives in the library.

---

## The catalog grows with usage

Every new KEL primitive is a note. A note is a Go function, registered in a `FuncMap`, documented in a markdown file, discoverable via `ork notes search`. When the function ships, it is available everywhere the engine runs. When the doc ships, it appears in the CLI and the reference docs.

Today the catalog has 257 notes across 30 domains: strings, math, time, conditions, replicas, pods, jobs, services, ingress, HPA, PVC, Prometheus, StatefulSets, and more. Every domain is a cluster of knowledge — the accumulated understanding of how to reason about that part of the Kubernetes API.

Nobody needs to write `hasCrashingPod` again.

And you don't have to wait for the catalog to grow. The `notes:` block in any Katalog lets your team define their own primitives — expressions that encode your domain knowledge, your naming conventions, your operational policies. They compose with every built-in note. They are available in every template expression in that Katalog. If a primitive you define turns out to be universally useful, it gets contributed to the core catalog and the whole community inherits it.

This is how a shared vocabulary grows: teams solve their own problems, the good solutions surface, and the knowledge stops being private.

---

## What comes next

KEL is currently specific to Orkestra. The template engine, the function registration, the catalog — all live inside the Orkestra binary. But the design is separable. The notes package has no dependency on the Orkestra runtime. It is a pure function library that happens to be injected into Orkestra's template engine.

The logical extension: publish the catalog as a standalone Go module. Any framework that uses `text/template` or `html/template` can import it and get Kubernetes expression capabilities immediately. The catalog becomes a shared commons — not owned by any one framework, not reinvented by any one team.

The idea is not new. npm did this for JavaScript. PyPI did it for Python. The Kubernetes ecosystem deserves a composable expression library that grows with community knowledge and is available anywhere a template is evaluated.

We called it KEL because it is specific to Kubernetes, extends the template language rather than replacing it, and carries the knowledge that should have been in a library from the beginning.

---

*The catalog is in [`pkg/note/`](https://github.com/orkspace/orkestra/tree/main/pkg/note). The reference is at [`ork notes`](https://github.com/orkspace/orkestra/blob/main/cmd/cli/notes.go). The primitives are yours to compose.*

→ [Browse the full note catalog](../reference/orkestra-notes/index.md)
