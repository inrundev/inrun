package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/external"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	"github.com/orkspace/orkestra/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// httpDoer is satisfied by *http.Client and any test double.
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// remoteRequest is the JSON body POSTed to the remote endpoint.
type remoteRequest struct {
	Key      string                  `json:"key"`
	GVK      schema.GroupVersionKind `json:"gvk"`
	Object   interface{}             `json:"object"`
	Args     map[string]interface{}  `json:"args,omitempty"`
	Prepared interface{}             `json:"prepared"`
}

// callRemote POSTs the reconcile request to the remote endpoint and returns the decoded result.
func (r *RemoteReconciler) callRemote(ctx context.Context, req domain.Request) (RemoteReconcileResult, error) {
	endpoint := r.resolveEndpoint(req)

	body, err := r.buildBody(req)
	if err != nil {
		return RemoteReconcileResult{}, fmt.Errorf("remote reconciler: build request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return RemoteReconcileResult{}, fmt.Errorf("remote reconciler: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if err := r.applyAuth(ctx, httpReq); err != nil {
		return RemoteReconcileResult{}, fmt.Errorf("remote reconciler: auth: %w", err)
	}

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return RemoteReconcileResult{}, fmt.Errorf("remote reconciler: POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RemoteReconcileResult{}, fmt.Errorf("remote reconciler: POST %s: unexpected status %d", endpoint, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return RemoteReconcileResult{}, fmt.Errorf("remote reconciler: read response: %w", err)
	}

	var result RemoteReconcileResult
	if err := json.Unmarshal(data, &result); err != nil {
		return RemoteReconcileResult{}, fmt.Errorf("remote reconciler: decode response: %w", err)
	}
	return result, nil
}

// resolveEndpoint evaluates the endpoint as a template when it contains "{{".
// Falls back to the static string on failure or when no resolver is available.
func (r *RemoteReconciler) resolveEndpoint(req domain.Request) string {
	if req.Prepared == nil || req.Prepared.Context == nil {
		return r.decl.Endpoint
	}
	resolver, ok := req.Prepared.Context.(*orktmpl.Resolver)
	if !ok {
		return r.decl.Endpoint
	}
	eval := resolver.TemplateEvaluator()
	resolved, ok := eval(r.decl.Endpoint)
	if !ok {
		return r.decl.Endpoint
	}
	return resolved
}

// buildBody serialises the remoteRequest payload.
func (r *RemoteReconciler) buildBody(req domain.Request) ([]byte, error) {
	var obj interface{}
	var prepared interface{}
	if req.Prepared != nil {
		obj = req.Prepared.Object
		if r.decl.Payload != nil && r.decl.Payload.Object != nil && len(r.decl.Payload.Object.Exclude) > 0 {
			obj = r.applyObjectExclusions(req.Prepared.Object, r.decl.Payload.Object.Exclude)
		}
		prepared = &outboundPrepared{
			PreparedRequest: req.Prepared,
			Children:        r.listChildren(req.Prepared.Object),
		}
	}
	return json.Marshal(remoteRequest{
		Key:      req.Key,
		GVK:      r.gvk,
		Object:   obj,
		Args:     r.resolveArgs(req),
		Prepared: prepared,
	})
}

// applyObjectExclusions returns a deep copy of obj with the given dot-notation paths removed.
func (r *RemoteReconciler) applyObjectExclusions(obj domain.Object, paths []string) map[string]interface{} {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return domain.Raw(obj)
	}
	cp := u.DeepCopy().Object
	for _, path := range paths {
		utils.DeleteNestedPath(cp, path)
	}
	return cp
}

// resolveArgs evaluates template expressions in decl.Args against the CR's resolver context.
// Returns nil when no args are declared.
func (r *RemoteReconciler) resolveArgs(req domain.Request) map[string]interface{} {
	if len(r.decl.Args) == 0 {
		return nil
	}
	if req.Prepared == nil || req.Prepared.Context == nil {
		return r.decl.Args
	}
	resolver, ok := req.Prepared.Context.(*orktmpl.Resolver)
	if !ok {
		return r.decl.Args
	}
	return utils.ResolveArgsMap(r.decl.Args, resolver.TemplateEvaluator())
}

// applyAuth injects the credential into the request header when auth is declared.
func (r *RemoteReconciler) applyAuth(ctx context.Context, req *http.Request) error {
	if r.decl.Auth == nil {
		return nil
	}
	auth := r.decl.Auth
	if auth.SecretRef != nil && auth.SecretRef.Namespace == "" {
		cp := *auth.SecretRef
		cp.Namespace = r.namespace
		authCopy := *auth
		authCopy.SecretRef = &cp
		auth = &authCopy
	}
	credential, header, err := external.ResolveAuth(ctx, auth, r.kube.Clientset())
	if err != nil {
		return err
	}
	if credential != "" {
		req.Header.Set(header, "Bearer "+credential)
	}
	return nil
}
