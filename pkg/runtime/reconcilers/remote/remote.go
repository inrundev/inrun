package remote

import (
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orkhttp "github.com/orkspace/orkestra/pkg/runtime/reconcilers/remote/http"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// New returns the reconciler for the declared remote transport.
// Callers import only this package — transport selection is internal.
func New(
	decl *orktypes.RemoteReconcilerDeclaration,
	gvk schema.GroupVersionKind,
	kube kubeclient.Interface,
	ev event.Recorder,
	managedResources []domain.ManagedResource,
	ownNamespace string,
) (domain.Reconciler, error) {
	t := decl.Protocol
	if t == "" {
		t = orktypes.RemoteReconcileProtocolHTTP
	}
	switch t {
	case orktypes.RemoteReconcileProtocolHTTP:
		return orkhttp.New(decl, gvk, kube, ev, managedResources, ownNamespace), nil
	default:
		return nil, fmt.Errorf("remote reconciler: unsupported transport %q", t)
	}
}
