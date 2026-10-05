package simulate

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/runtime/reconcilers/remote"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// remoteNote tells the user the run depends on their reconciler process.
const remoteNote = "remote reconciler: each cycle calls the endpoint in reconcile.remote — start the reconciler before simulating"

// newRemoteReconciler builds the remote reconciler the runtime would use, over
// kube. Its HTTP calls go to the real endpoint; what it applies lands in kube.
func newRemoteReconciler(crd *orktypes.CRDEntry, kube kubeclient.Interface, ownNamespace string) (domain.Reconciler, error) {
	return remote.New(
		crd.Box().Reconcile.Remote,
		crd.GVK(),
		kube,
		event.Discard(),
		crd.RemoteManagedResources(),
		ownNamespace,
	)
}

// setupObjects reads the CRD's setup.apply files: the objects ork run applies
// before the operator starts, such as a remote reconciler's token Secret.
// The merger has already made the paths absolute.
func setupObjects(crd *orktypes.CRDEntry) ([]*unstructured.Unstructured, error) {
	if crd.Setup == nil {
		return nil, nil
	}
	var objs []*unstructured.Unstructured
	for _, entry := range crd.Setup.Apply {
		data, err := utils.ReadLocal(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("setup %s: %w", entry.Path, err)
		}
		dec := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
		for {
			u := &unstructured.Unstructured{}
			if err := dec.Decode(&u.Object); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, fmt.Errorf("setup %s: %w", entry.Path, err)
			}
			if u.GetKind() != "" {
				objs = append(objs, u)
			}
		}
	}
	return objs, nil
}
