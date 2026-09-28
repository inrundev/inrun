//go:build !runtime && !gateway

package simulate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/contract"
	"github.com/orkspace/orkestra/pkg/runtime/kordinator/prepare"
	"github.com/orkspace/orkestra/pkg/runtime/reconciler"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	envtestBinDir = ".ork/envtest-bins"
	// DefaultEnvtestK8sVersion is the Kubernetes version used when --envtest-version
	// is not set. "1.32" (two components) resolves to the latest stable 1.32.x patch
	// from the controller-runtime release index. The setup-envtest "1.32.x" wildcard
	// syntax is not accepted by the Go API — two-component form is equivalent.
	DefaultEnvtestK8sVersion = "1.32"
)

// RunWithEnvtest is identical to Run but starts a local kube-apiserver + etcd,
// installs the CRDs at crdPaths, applies the CR, then runs the shared loop
// against a real SharedIndexInformer watching the envtest API server.
//
// crdPaths must contain at least one CRD YAML — set via spec.crd / spec.crdFiles.
// Binaries are auto-downloaded to ~/.ork/envtest-bins on first run; KUBEBUILDER_ASSETS
// overrides this.
func RunWithEnvtest(ctx context.Context, kat *katalog.Katalog, crdName string,
	cr *unstructured.Unstructured, maxCycles int, opts RunOptions, crdPaths []string, k8sVersion string) (*Result, error) {

	prev := log.Logger
	log.Logger = log.Output(io.Discard)
	defer func() { log.Logger = prev }()

	// Suppress controller-runtime's own logger — it prints a noisy stack trace
	// if SetLogger is never called. Discarding is correct here: envtest runs
	// are short-lived and their internal logs are not part of simulate output.
	ctrl.SetLogger(logr.Discard())

	binDir := os.Getenv("HOME") + "/" + envtestBinDir

	env := &envtest.Environment{
		BinaryAssetsDirectory: binDir,
		CRDDirectoryPaths:     crdPaths,
		ErrorIfCRDPathMissing: true,
	}

	// Respect KUBEBUILDER_ASSETS; otherwise auto-download to ~/.ork/envtest-bins.
	if assets := os.Getenv("KUBEBUILDER_ASSETS"); assets != "" {
		env.BinaryAssetsDirectory = assets
	} else {
		env.DownloadBinaryAssets = true
		if k8sVersion == "" {
			k8sVersion = DefaultEnvtestK8sVersion
		}
		env.DownloadBinaryAssetsVersion = k8sVersion
		// Only print the download notice when the binaries don't yet exist locally.
		if _, err := os.Stat(binDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "envtest: downloading binaries to %s (one-time, subsequent runs use cache)\n", binDir)
		}
	}

	restCfg, err := env.Start()
	if err != nil {
		return nil, fmt.Errorf("envtest: starting API server: %w", err)
	}
	defer env.Stop() //nolint:errcheck

	crdEntry, ok := kat.CRDEntry(crdName)
	if !ok {
		return nil, fmt.Errorf("CRD %q not found in Katalog", crdName)
	}
	if crdEntry.WithRemoteDecl() {
		return nil, ErrRemoteReconciler
	}

	result := &Result{}
	box := effectiveOperatorBox(crdEntry, cr, opts.Target)
	for _, phase := range []*orktypes.HookTemplates{
		box.EffectiveOnCreate(),
		box.EffectiveOnReconcile(),
		box.EffectiveOnDelete(),
	} {
		if phase == nil {
			continue
		}
		filtered, skipped := orktypes.FilterSimulatable(*phase)
		*phase = filtered
		result.Notes = append(result.Notes, skipped...)
	}

	scheme, err := kat.Scheme()
	if err != nil {
		return nil, fmt.Errorf("building scheme: %w", err)
	}

	// Attach the recording transport BEFORE building the kubeclient so that ALL
	// API calls (typed clientset, dynamic client, controller-runtime client) go
	// through the interceptor. The fake simulate path uses reactor chains; in
	// envtest there are no reactors, so transport-level interception is the only
	// reliable way to capture every write op regardless of which client is used.
	shared := &sharedOps{}
	restCfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return &recordingTransport{inner: rt, shared: shared}
	}

	realKube, err := kubeclient.NewKubeclientFromConfig(ctx, restCfg, scheme)
	if err != nil {
		return nil, fmt.Errorf("building kubeclient: %w", err)
	}

	// managedKube wires the real kubeclient to the shared op store so that
	// loopKube methods (AdvanceCycle, OpsForCycle, etc.) work correctly.
	recKube := &managedKube{Interface: realKube, shared: shared}

	// Apply the CR to the real API server before the loop starts.
	gvr := crdEntry.GVR()
	crData, err := json.Marshal(cr.Object)
	if err != nil {
		return nil, fmt.Errorf("marshalling CR: %w", err)
	}
	var crObj unstructured.Unstructured
	if err := json.Unmarshal(crData, &crObj.Object); err != nil {
		return nil, fmt.Errorf("copying CR: %w", err)
	}
	ns := crObj.GetNamespace()
	if ns == "" {
		_, err = recKube.DynamicClient().Resource(gvr).Create(ctx, &crObj, metav1.CreateOptions{})
	} else {
		_, err = recKube.DynamicClient().Resource(gvr).Namespace(ns).Create(ctx, &crObj, metav1.CreateOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("seeding CR into envtest: %w", err)
	}

	// Discard the CR-seed op — it is infrastructure, not a reconcile output.
	shared.mu.Lock()
	shared.ops = shared.ops[:0]
	shared.mu.Unlock()

	seedManagedMeta(cr, kat.Metadata().Name)

	gvk := schema.GroupVersionKind{
		Group:   crdEntry.APITypes.Group,
		Version: crdEntry.APITypes.Version,
		Kind:    crdEntry.APITypes.Kind,
	}

	newObjFn := func() domain.Object { return &unstructured.Unstructured{} }
	if objFactory, ok := orktypes.ObjectRegistry[gvk]; ok {
		if _, ok := objFactory().(domain.Object); ok {
			newObjFn = func() domain.Object {
				return orktypes.ObjectRegistry[gvk]().(domain.Object)
			}
		}
	}

	// Build a real SharedIndexInformer backed by the envtest API server via a
	// dynamic ListWatch. This gives the reconciler real watch stream delivery and
	// real object state between cycles — unlike the static fakeInformer.
	lw := &cache.ListWatch{
		ListFunc: func(lopts metav1.ListOptions) (k8sruntime.Object, error) {
			if ns == "" {
				return recKube.DynamicClient().Resource(gvr).List(ctx, lopts)
			}
			return recKube.DynamicClient().Resource(gvr).Namespace(ns).List(ctx, lopts)
		},
		WatchFunc: func(lopts metav1.ListOptions) (watch.Interface, error) {
			if ns == "" {
				return recKube.DynamicClient().Resource(gvr).Watch(ctx, lopts)
			}
			return recKube.DynamicClient().Resource(gvr).Namespace(ns).Watch(ctx, lopts)
		},
	}
	inf := cache.NewSharedIndexInformer(lw, &unstructured.Unstructured{}, 0, cache.Indexers{})
	stopCh := make(chan struct{})
	defer close(stopCh)
	go inf.Run(stopCh)
	if !cache.WaitForCacheSync(ctx.Done(), inf.HasSynced) {
		return nil, fmt.Errorf("envtest: informer cache sync timed out")
	}

	var hookBinder domain.AnyReconcileHooks
	if fn, ok := orktypes.HookRegistry[gvk]; ok {
		hookBinder = fn()
	}

	peerRegistry := kordinator.NewKordinatorRegistry()
	for _, peerName := range kat.CRDNames() {
		if peerName == crdEntry.Name {
			continue
		}
		peerEntry, ok := kat.CRDEntry(peerName)
		if !ok {
			continue
		}
		peerCR, ok := opts.Peers[strings.ToLower(peerEntry.APITypes.Kind)]
		if !ok {
			continue
		}
		peerIdx := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
		_ = peerIdx.Add(peerCR)
		peerInf := newFakeInformer(peerIdx)
		peerGVKStr := schema.GroupVersionKind{
			Group:   peerEntry.APITypes.Group,
			Version: peerEntry.APITypes.Version,
			Kind:    peerEntry.APITypes.Kind,
		}.String()
		peerRegistry.Register(peerGVKStr, peerEntry, peerInf, nil)
	}

	var r domain.Reconciler
	if factoryFn, ok := orktypes.ReconcilerRegistry[gvk]; ok {
		r = factoryFn(recKube.WithInformer(inf).WithEventRecorder(event.Discard()))
	} else {
		r = reconciler.NewGenericReconciler(
			crdEntry,
			inf,
			nil,
			recKube,
			hookBinder,
			newObjFn,
			nil, nil,
			kat,
		)
	}

	key, err := resolveCacheKey(cr)
	if err != nil {
		return nil, err
	}

	r = wrapWithGate(r, box.PreReconcile, kat.Notes, getFromIndexerOrFallback(inf.GetIndexer(), key, cr))

	prepInput := prepare.Input{
		Entry: contract.RegistryEntry{
			CRD:      crdEntry,
			Informer: inf,
		},
		Key:      key,
		Kat:      kat,
		Kube:     recKube,
		Registry: peerRegistry,
	}

	loopResult := runLoop(ctx, r, recKube, key, maxCycles, prepInput)
	loopResult.Notes = result.Notes
	return loopResult, nil
}

// ── HTTP transport recording ────────────────────────────────────────────────
//
// recordingTransport wraps the real HTTP transport and intercepts every
// successful write request to record an Op. This captures ALL kubeclient paths
// (typed clientset, dynamic client, controller-runtime client) without needing
// reactor chains, which only work on fake clients.

type recordingTransport struct {
	inner  http.RoundTripper
	shared *sharedOps
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.inner.RoundTrip(req)
	if err != nil || resp == nil || !isWriteMethod(req.Method) || resp.StatusCode >= 300 {
		return resp, err
	}

	verb, resource, namespace, name := parseAPIRequest(req)

	// POST creates: name is absent from the collection URL — read it from the
	// response body, then restore the body for the caller.
	if req.Method == http.MethodPost && name == "" {
		resource, namespace = parseCollectionURL(req)
		if resource != "" {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(body))
			if readErr == nil {
				name = extractMetaName(body)
				verb = "create"
			}
		}
	}

	if resource != "" && name != "" {
		rt.shared.mu.Lock()
		rt.shared.ops = append(rt.shared.ops, Op{
			Cycle:     rt.shared.cycle,
			Verb:      verb,
			Resource:  resource,
			Namespace: namespace,
			Name:      name,
			At:        time.Now(),
		})
		rt.shared.mu.Unlock()
	}
	return resp, err
}

func isWriteMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodPatch || method == http.MethodDelete
}

// parseAPIRequest extracts (verb, resource, namespace, name) from a Kubernetes
// API request. Returns empty strings for requests that don't target a named object
// (e.g. list, watch) or for subresource paths we intentionally skip.
//
// Supported URL formats:
//
//	/api/v1/namespaces/{ns}/{resource}/{name}[/{subresource}]
//	/api/v1/{resource}/{name}[/{subresource}]
//	/apis/{group}/{version}/namespaces/{ns}/{resource}/{name}[/{subresource}]
//	/apis/{group}/{version}/{resource}/{name}[/{subresource}]
func parseAPIRequest(req *http.Request) (verb, resource, namespace, name string) {
	p := strings.TrimPrefix(req.URL.Path, "/")
	parts := strings.Split(p, "/")

	var res, ns, n string
	switch {
	case len(parts) >= 5 && parts[0] == "api" && parts[2] == "namespaces":
		// /api/v1/namespaces/{ns}/{resource}/{name}[/subresource]
		ns, res, n = parts[3], parts[4], ""
		if len(parts) >= 6 {
			n = parts[5]
		}
	case len(parts) >= 4 && parts[0] == "api":
		// /api/v1/{resource}/{name}
		res, n = parts[2], ""
		if len(parts) >= 4 {
			n = parts[3]
		}
	case len(parts) >= 7 && parts[0] == "apis" && parts[3] == "namespaces":
		// /apis/{group}/{version}/namespaces/{ns}/{resource}/{name}[/subresource]
		ns, res, n = parts[4], parts[5], ""
		if len(parts) >= 7 {
			n = parts[6]
		}
	case len(parts) >= 5 && parts[0] == "apis":
		// /apis/{group}/{version}/{resource}/{name}
		res, n = parts[3], ""
		if len(parts) >= 5 {
			n = parts[4]
		}
	}

	if n == "" || strings.Contains(n, "?") {
		return
	}

	// Map HTTP method + content type to Op verb.
	ct := req.Header.Get("Content-Type")
	switch req.Method {
	case http.MethodPost:
		verb = "create"
	case http.MethodPut:
		verb = "update"
	case http.MethodDelete:
		verb = "delete"
	case http.MethodPatch:
		if strings.Contains(ct, "apply-patch") {
			verb = "apply"
		} else {
			verb = "patch"
		}
	}

	resource, namespace, name = res, ns, n
	return
}

// ── loopKube adapter ─────────────────────────────────────────────────────────

type sharedOps struct {
	mu    sync.Mutex
	ops   []Op
	cycle int
}

// managedKube adapts a real kubeclient.Interface to the loopKube interface by
// wiring AdvanceCycle / OpsForCycle / Ops / MarkDeploymentReady to the shared op
// store, which is populated by the recording HTTP transport.
type managedKube struct {
	kubeclient.Interface
	shared *sharedOps
}

func (m *managedKube) AdvanceCycle() {
	m.shared.mu.Lock()
	m.shared.cycle++
	m.shared.mu.Unlock()
}

func (m *managedKube) OpsForCycle(cycle int) []Op {
	m.shared.mu.Lock()
	defer m.shared.mu.Unlock()
	var out []Op
	for _, op := range m.shared.ops {
		if op.Cycle == cycle {
			out = append(out, op)
		}
	}
	return out
}

func (m *managedKube) Ops() []Op {
	m.shared.mu.Lock()
	defer m.shared.mu.Unlock()
	out := make([]Op, len(m.shared.ops))
	copy(out, m.shared.ops)
	return out
}

func (m *managedKube) MarkDeploymentReady(namespace, name string) {
	dep, err := m.Interface.Clientset().AppsV1().Deployments(namespace).Get(
		context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return
	}
	if dep.Spec.Replicas != nil {
		dep.Status.AvailableReplicas = *dep.Spec.Replicas
		dep.Status.ReadyReplicas = *dep.Spec.Replicas
		dep.Status.Replicas = *dep.Spec.Replicas
	}
	m.Interface.Clientset().AppsV1().Deployments(namespace).UpdateStatus( //nolint:errcheck
		context.Background(), dep, metav1.UpdateOptions{})
}

// compile-time check.
var _ loopKube = (*managedKube)(nil)
