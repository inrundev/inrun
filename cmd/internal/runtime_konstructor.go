// konstructRuntime assembles the complete Orkestra runtime registry.
//
// This is the dependency-injection boundary for the runtime: komponents are
// constructed here and their dependencies are wired here. Nothing is started
// here; startup is handled by orkestra.Start() in declaration order.
//
// The resulting registry contains the Katalog, Kubernetes clients and
// informers, resource registries, Kordinator, and health server.
package internal

import (
	"context"
	"strings"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/health"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/katalog/pipeline"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/orkspace/orkestra/pkg/process"
	"github.com/orkspace/orkestra/pkg/runtime/informer"
	"github.com/orkspace/orkestra/pkg/runtime/informer/observe"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/vitals"
	"github.com/orkspace/orkestra/pkg/runtime/queue"
	"github.com/orkspace/orkestra/pkg/runtime/reconcilers/generic"
	"github.com/orkspace/orkestra/pkg/runtime/reconcilers/mux"
	"github.com/orkspace/orkestra/pkg/runtime/reconcilers/remote"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/version"
	"k8s.io/client-go/tools/cache"
)

// runtimeKfg is the assembled runtime — returned to runtime.go so it can call
// orkestra.Start(ctx) and block until shutdown.
type runtimeKfg struct {
	konfig   *konfig.Konfig
	katalog  *katalog.Katalog
	komp     *[]domain.Komponent
	event    *event.Event
	kube     *kubeclient.Kubeclient
	kord     *kordinator.DependencyKordinator
	orkestra *process.Manager
}

// konstructRuntime wires the entire Orkestra runtime.
//
// Nothing is started here. Every component is constructed and threaded together
// as closures and pointers. orkestra.Start() calls komponent.Start() in
// registration order, and komponent.Stop() in reverse order on shutdown.
//
// The method is intentionally long — this is the one place where all wiring
// is visible. Splitting it would scatter the dependency graph across files
// and make it harder to reason about startup order.
func konstructRuntime(kfg *konfig.Konfig, m *merger.Merger, ctx context.Context) *runtimeKfg {

	// ── 0. Runtime facts — available to all template expressions as .ork.* ──────
	orkestraNamespace := kfg.Cluster().Namespace()
	ctx = orktmpl.ContextWithOrkContext(ctx, orktmpl.NewOrkContext(
		kfg.Cluster().Namespace(),
		version.Version,
	))

	// ── 1a. Instance ────────────────────────────────────────────────────────────
	kfg.SetInstance(konfig.Runtime())

	// ── 1b. Katalog ────────────────────────────────────────────────────────────
	// Loads and validates the YAML Katalog. After this point, kat.Enabled()
	// returns only CRDs that passed schema validation and are not disabled.
	// Invalid CRDs are logged and excluded — they do not block the operator.
	kat := pipeline.NewKatalog(kfg, m)

	if registryURL := kfg.RegistryConfig().RegistryURL; registryURL != "" {
		m.SetRegistryURL(registryURL)
		logger.Info().Str("registry", registryURL).Msg("registry URL configured from ORK_REGISTRY")
	}

	// ── 2. Scheme ─────────────────────────────────────────────────────────────
	// Each CRD type (e.g. *PipelineList) must be registered with the scheme so
	// the REST client knows how to decode API server responses. For dynamic CRDs
	// (unstructured mode), this is a no-op — they use the dynamic client.
	scheme, err := katalog.NewSchemeRegistry(kat)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to build scheme registry")
	}

	// ── 3. Core komponents ────────────────────────────────────────────────────
	// Created here, started later by orkestra in registration order.

	kube := kubeclient.NewKubeclient(kfg, scheme)

	// kubeclient is started immediately — the informer factory's missing-CRD
	// check needs the REST config during construction, before orkestra.Start().
	if err := kube.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start kubeclient")
	}

	// Clientset is built inside Start; capture after Start so cs is never nil.
	cs := kube.Clientset()

	// HealthServer — HTTP-only (health, readiness, metrics, Katalog API routes).
	// Routes registered below before Start() binds the port.
	hs := health.NewHealthServer(kfg)

	// Event recorder — surfaces notable state changes to the Kubernetes event stream
	// (visible via kubectl describe). Shared by all controllers that emit events.
	ev := event.NewEvent(kube)

	// Default work queue — rate-limiting reconcile queue shared by controllers that
	// do not need a dedicated queue. Bounded to prevent runaway reconcile storms.
	defaultWq := queue.NewWorkqueue("default workqueue")

	// Queue registry — maps GVK strings to dedicated work queues. Controllers
	// register here so that cross-controller enqueues are dispatched correctly.
	queueRegistry := queue.NewQueueRegistry()

	// ── 4a. REST client provider ──────────────────────────────────────────────
	// Associates each CRD type with a constructor that builds a typed REST
	// client. The constructor is deferred — called on first informer use.
	// Dynamic CRDs skip this — they use the dynamic client directly.
	provider := kube.NewClientProvider()

	for _, crd := range kat.Enabled() {
		crd := crd
		if crd.IsDynamic() {
			continue
		}
		object, list := crd.GetRuntimeObjects()
		logger.Debug().Str("gvk", crd.GVKString()).Msg("registering CRD client provider")

		provider.Register(object, func(k *kubeclient.Kubeclient) (informer.GenericClient, error) {
			return k.NewClient(list, kubeclient.CRDInfo{
				Kind:          crd.APITypes.Kind,
				Group:         crd.APITypes.Group,
				Version:       crd.APITypes.Version,
				APIPath:       crd.APITypes.APIPath,
				GroupVersion:  crd.GroupVersion,
				Plural:        crd.APITypes.Plural,
				Namespace:     crd.Namespace,
				Namespaced:    crd.IsNamespaced(),
				ForceConflict: crd.ResolveForceConflict(),
			})
		})
	}

	// ── 4b. Shared informer factory ───────────────────────────────────────────
	// Creates one SharedIndexInformer per CRD. On Start(), each informer opens
	// a watch against the API server and populates its in-memory cache.
	// Watch events are routed into per-CRD workqueues via handleEvent.
	infFactory := informer.SharedInformerFactory(kube.RestConfig(), informer.FactoryOptions{
		Provider:      provider,
		Scheme:        scheme,
		DefaultWq:     defaultWq,
		Konfig:        kfg,
		Katalog:       kat,
		ClientSet:     cs,
		QueueRegistry: queueRegistry,
	})

	// ── 4d. Kordinator registry + per-CRD wiring ──────────────────────────────
	// ktrlRegistry maps GVK → (CRDEntry, SharedIndexInformer, ReconcilerFactory).
	// It also implements reconciler.KatalogRegistry via GetInformerByName,
	// enabling cross-CRD observation with zero API server calls.
	ktrlRegistry := kordinator.NewKordinatorRegistry()

	// ── 4e. CRD health map ────────────────────────────────────────────────────
	// One CRDHealth per CRD — shared between the DependencyKordinator
	// (which updates it on each reconcile) and the HTTP health routes
	// (which read it on each request). All three reference the same pointers.
	crdHealthMap := make(map[string]*vitals.CRDHealth)
	for _, crd := range kat.Enabled() {
		gvk := crd.GVKString()
		crdHealthMap[gvk] = vitals.NewCRDHealth(crd.Name)
	}

	logger.Debug().Msg("wiring CRDs into kordinator registry...")

	finalizers := kfg.Finalizers()
	for _, crd := range kat.Enabled() {
		crd := crd
		gvk := crd.GVKString()
		object, _ := crd.GetRuntimeObjects()

		wq := queueRegistry.Register(gvk, crd.QueueConfig())

		// compute selectors
		labelSelector := orktypes.Labels(crd.LabelSelector).String()
		fieldSelector := orktypes.Labels(crd.FieldSelector).String()

		opts := informer.Options{
			Name:          crd.APITypes.Kind,
			Resync:        crd.SetResync(0),
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		}
		if crd.SharedQueue() {
			opts.Wq = nil // use the shared default queue
		} else {
			opts.Wq = wq
		}

		// ── Namespace filter — Tier 1 (scope ListerWatcher) + Tier 2 (pre-enqueue) ──
		// Tier 2 is always registered when namespace rules exist.
		// Tier 1 scopes the ListerWatcher to a single namespace when allowedNamespaces
		// has exactly one entry — the informer never sees events from other namespaces.
		if crd.HasNamespaceRules() {
			if crd.IsSingleNamespace() {
				opts.Namespace = crd.SingleNamespace()
				logger.Debug().
					Str("crd", crd.APITypes.Kind).
					Str("namespace", opts.Namespace).
					Msg("informer: namespace-scoped watch (Tier 1)")
			}
			filter := &informer.NamespaceFilter{
				AllowedNamespaces:    []string(crd.AllAllowedNamespaces()),
				RestrictedNamespaces: []string(crd.AllRestrictedNamespaces()),
			}
			infFactory.RegisterNamespaceFilter(gvk, filter)
			logger.Debug().
				Str("crd", crd.APITypes.Kind).
				Str("filter", informer.NamespaceFilterSummary(filter)).
				Msg("informer: namespace filter registered (Tier 2)")
		}

		// Choose typed or dynamic informer.
		// Dynamic CRDs use *unstructured.Unstructured — no Go type needed.
		// Typed CRDs use the registered concrete Go type for type-safe access.
		var inf cache.SharedIndexInformer

		// For dynamic CRDs, use opts.Namespace from the Tier 1 filter when set;
		// otherwise fall back to crd.Namespace (operator-level namespace setting).
		dynNamespace := crd.Namespace
		if opts.Namespace != "" {
			dynNamespace = opts.Namespace
		}

		logger.Debug().
			Bool("dynamic:", crd.IsDynamic()).
			Msgf("[DEBUG] CRD %s: location = %q\n", crd.APITypes.Kind, crd.APITypes.Location)
		if crd.IsDynamic() {
			lw := kube.NewDynamicListerWatcher(kubeclient.CRDInfo{
				Kind:          crd.APITypes.Kind,
				Group:         crd.APITypes.Group,
				Version:       crd.APITypes.Version,
				APIPath:       crd.APITypes.APIPath,
				GroupVersion:  crd.GroupVersion,
				Plural:        crd.APITypes.Plural,
				Namespace:     dynNamespace,
				Namespaced:    crd.IsNamespaced(),
				ForceConflict: crd.ResolveForceConflict(),
			}, kubeclient.ListOptions{
				LabelSelector: labelSelector,
				FieldSelector: fieldSelector,
			})
			inf = infFactory.ForListerWatcher(lw, object, ctx, opts)
		} else {
			inf = infFactory.For(object, ctx, opts)
		}

		finalizers = append(finalizers, crd.Box().EffectiveFinalizers()...)

		infCopy := inf

		// Build the reconciler factory.
		// For default: true CRDs — generic.Reconciler interprets the Katalog declaratively.
		// For default: false CRDs — a custom Constructor is required.
		//
		// The factory is a closure — it captures all values at construction time
		// and is called by startCRDWorkers after informers are synced.
		// Each call returns a fresh reconciler instance for one worker goroutine.
		var factory func() domain.Reconciler

		if crd.DefaultReconcile() {
			objCopy := object

			var anyHooks domain.AnyReconcileHooks
			if r := crd.Box().Reconcile; r != nil && r.HookFactory != nil {
				anyHooks = r.HookFactory()
			}

			logger.Debug().Str("gvk", gvk).Msg("wiring generic.Reconciler factory")

			// Attach hooks.args to a copy of the kube client; hooks read them via kube.Args().
			var hookKube kubeclient.Interface = kube.
				WithForceConflict(crd.ResolveForceConflict())
			if args := crd.HooksArgs(); len(args) > 0 {
				hookKube = kube.WithArgs(kubeclient.Args(args))
			}

			factory = func() domain.Reconciler {
				return generic.New(
					crd,
					infCopy,
					ev,
					hookKube,
					anyHooks,
					func() domain.Object {
						return objCopy.DeepCopyObject().(domain.Object)
					},
					kat,
				)
			}
		} else if crd.WithRemoteDecl() {
			logger.Debug().Str("gvk", gvk).Msg("wiring RemoteReconciler factory")

			remoteDecl := crd.Box().Reconcile.Remote
			crdManagedResources := crd.RemoteManagedResources()
			remoteKube := kube.WithStoreFor(infFactory.StoreFor)
			factory = func() domain.Reconciler {
				r, err := remote.New(remoteDecl, crd.GVK(), remoteKube, ev, crdManagedResources, orkestraNamespace)
				if err != nil {
					logger.Fatal().Err(err).Str("gvk", gvk).Msg("failed to build remote reconciler")
				}
				return r
			}
		} else {
			if !crd.ConstructorEnabled() {
				logger.Fatal().
					Str("gvk", gvk).
					Msg("reconciler.default is false but no Constructor provided")
			}

			logger.Debug().Str("gvk", gvk).Msg("wiring custom reconciler factory")

			// Attach constructor.args, informer, and event recorder to a copy of the
			// kube client. Constructor authors access them via kube.GetInformer() etc.
			var ctorKube kubeclient.Interface = kube.
				WithInformer(infCopy).
				WithEventRecorder(ev).
				WithStoreFor(infFactory.StoreFor).
				WithIndexerFor(infFactory.IndexerFor).
				WithForceConflict(crd.ResolveForceConflict())
			if args := crd.ConstructorArgs(); len(args) > 0 {
				ctorKube = ctorKube.WithArgs(kubeclient.Args(args))
			}

			factory = func() domain.Reconciler {
				return crd.Box().Reconcile.Constructor(ctorKube)
			}
		}

		// Wrap with the mux reconciler when per-target constructors or remote reconcilers
		// are declared. The mux reconciler dispatches each reconcile cycle to the matching
		// target's domain.Reconciler, falling back to the base factory for CRs with
		// no annotation or an unrecognised target name.
		if crd.HasTargetConstructorFactories() || crd.HasTargetRemoteDeclarations() {
			baseFactory := factory
			crdCopy := crd
			factory = func() domain.Reconciler {
				ctorCount := len(crdCopy.TargetReconcilerFactories)
				remoteDecls := crdCopy.TargetRemoteDeclarations()
				targets := make(map[string]domain.Reconciler, ctorCount+len(remoteDecls))
				for targetName, ctor := range crdCopy.TargetReconcilerFactories {
					var targetKube kubeclient.Interface = kube.
						WithInformer(infCopy).
						WithEventRecorder(ev).
						WithStoreFor(infFactory.StoreFor).
						WithIndexerFor(infFactory.IndexerFor).
						WithForceConflict(crdCopy.ResolveForceConflict())
					if args := crdCopy.TargetConstructorArgs(targetName); len(args) > 0 {
						targetKube = targetKube.WithArgs(kubeclient.Args(args))
					}
					targets[targetName] = ctor(targetKube)
				}
				for targetName, remoteDecl := range remoteDecls {
					r, err := remote.New(remoteDecl, crdCopy.GVK(), kube.WithStoreFor(infFactory.StoreFor), ev, crdCopy.RemoteManagedResources(), orkestraNamespace)
					if err != nil {
						logger.Fatal().Err(err).Str("gvk", gvk).Str("target", targetName).Msg("failed to build remote reconciler")
					}
					targets[targetName] = r
				}
				return mux.New(infCopy, targets, baseFactory())
			}
			logger.Debug().
				Str("gvk", gvk).
				Int("constructorTargets", len(crd.TargetReconcilerFactories)).
				Int("remoteTargets", len(crd.TargetRemoteDeclarations())).
				Msg("wiring mux reconciler factory")
		}

		// Register informs the DependencyKordinator which informer and factory
		// belong to this CRD. Workers are not started yet — that happens in Start().
		ktrlRegistry.Register(gvk, crd, inf, factory)
		logger.Debug().Str("gvk", gvk).Msg("CRD registered")
	}

	// ── 5. HTTP routes ───────────────────────────────────────────────────────
	// All routes registered before hs.Start() — the mux is shared.
	//
	// Per-CRD routes:
	//   /katalog/{crd}/health       		→ 200 healthy, 503 degraded
	//   /katalog/{crd}              		→ CRD config + live reconcile stats
	//	 /katalog/{crd}/raw			 		→ the user's config
	//   /katalog/{crd}/enriched	 		→ the runtime config
	//   /katalog/{crd}/cr           		→ all CR instances (informer cache, <1ms)
	//   /katalog/{crd}/cr/{ns}/{n}  		→ CR detail + children (watch cache, <50ms)
	//   /katalog/{crd}/cr/{...}/events 	→ recent events (watch cache, <50ms)
	//
	// Aggregate:
	//	 /katalog/raw				 		→ the user's katalog config
	//	 /katalog/enriched				 	→ the runtime katalog config
	//   /katalog                    		→ all CRDs, dependency graph, health summary
	orkHealth := vitals.NewRuntimeHealth()

	for _, crd := range kat.Enabled() {
		gvk := crd.GVKString()
		crdHealth := crdHealthMap[gvk]
		crdName := strings.ToLower(crd.Name)

		entry, _ := ktrlRegistry.Get(gvk)
		inf := entry.Informer

		if !crd.IsEnabledAllEndpoints() {
			continue
		}

		if crd.IsHealthEnabled() {
			hs.Register(
				"/katalog/"+crdName+"/health",
				vitals.BuildCRDHealthHandler(crd, kfg, inf, crdHealth, orkHealth),
			)
		}

		if crd.IsInfoEnabled() {
			hs.Register(
				"/katalog/"+crdName,
				vitals.BuildCRDInfoHandler(
					crd, kfg, inf, crdHealth,
					orkHealth,
				),
			)
			hs.Register(
				"/katalog/"+crdName+"/cr",
				vitals.BuildCRListHandler(crd, inf, orkHealth),
			)
			hs.Register(
				"/katalog/"+crdName+"/cr/",
				vitals.BuildCRDetailAndEventsHandler(crd, inf, kube, crd.Box(), orkHealth),
			)
		}

		// Register raw and enriched CRD definition endpoint
		hs.Register(
			"/katalog/"+crdName+"/raw",
			vitals.BuildCRDRawHandler(m, crd.Name),
		)
		hs.Register(
			"/katalog/"+crdName+"/enriched",
			vitals.BuildCRDEnrichedHandler(kat, crd.Name),
		)

		logger.Debug().
			Str("health", "/katalog/"+crdName+"/health").
			Str("info", "/katalog/"+crdName).
			Str("raw", "/katalog/"+crdName+"/raw").
			Str("enriched", "/katalog/"+crdName+"/enriched").
			Msg("registered CRD routes")
	}

	hs.Register("/katalog/raw", vitals.BuildRawKatalogHandler(m))
	hs.Register("/katalog/enriched", vitals.BuildEnrichedKatalogHandler(kat))
	hs.Register("/katalog", vitals.BuildKatalogHandler(kat, kfg, ktrlRegistry, crdHealthMap, orkHealth))

	// ── 6a. Secondary resource observers ────────────────────────────
	// Observe secondary resources declared in operatorBox.observe.watch/events.
	obs := observe.New(observe.Dependencies{
		Kube:          kube,
		Informer:      infFactory,
		QueueRegistry: queueRegistry,
		Katalog:       kat,
	})

	// ── 6b. Dependency kordinator ──────────────────────────────────────────────
	// Starts CRD workers in topological order defined by the dependency graph.
	// For each CRD, waits until all declared dependsOn CRDs meet their
	// condition (started | healthy) before calling factory() and starting workers.
	//
	// Worker lifecycle:
	//   Start()  → wait for informer sync → call factory() per worker → run loop
	//   Shutdown → drain queue → stop workers → remove from active set
	kord := kordinator.NewDependencyKordinator(
		kube,
		infFactory,
		obs,
		ktrlRegistry,
		kat,
		ev,
		hs,
		queueRegistry,
		defaultWq,
		crdHealthMap,
		orkHealth,
		kfg.Katalog().DefaultWorkers(),
		katalog.NewDependencyGraph(kat),
		kfg.Katalog().ShutdownTimeout(),
	)

	// ── 7. Komponent list ─────────────────────────────────────────────────────
	// Start order: each komponent must start after its dependencies.
	// Stop order: reverse of start order (automatic).
	//
	// HealthServer starts first so it can serve /ready during startup.
	// Kubeclient is already started above but is still registered so
	// orkestra manages its Stop().
	komponents := []domain.Komponent{
		hs,            // 1. HTTP server — /ready, /livez, /katalog routes
		kube,          // 2. REST clients — already started, managed for Stop()
		ev,            // 3. event recorder — depends on kube
		queueRegistry, // 4. per-CRD bounded queues
		defaultWq,     // 5. default unbounded queue
		infFactory,    // 6. informer factory — starts watchers, closes ready channel
		kord,          // 7. dependency kordinator — starts workers in topo order
	}

	// suppress unused variable warning — finalizers is populated but only
	// referenced by the reconciler closures captured above.
	_ = finalizers

	// ── 8. Orkestra ───────────────────────────────────────────────────────────
	// The supervisor. Calls Start() on each komponent in order.
	// On OS signal (SIGTERM/SIGINT) or fatal error, calls Stop() in reverse.
	// Graceful shutdown: drains queues before stopping workers.
	o := process.New(
		kfg.RunningInstance(),
		kfg.Katalog().ShutdownGracePeriod(),
		kfg.Ork().LogLevel(),
	)
	o.Register(komponents)

	return &runtimeKfg{
		konfig:   kfg,
		katalog:  kat,
		komp:     &komponents,
		event:    ev,
		kube:     kube,
		kord:     kord,
		orkestra: o,
	}
}
