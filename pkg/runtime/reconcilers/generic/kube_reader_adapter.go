// pkg/reconciler/kube_reader_adapter.go
//
// KubeReader adapter — bridges kubeclient.Kubeclient to the orktypes.KubeReader
// interface used by provider libraries.
//
// Providers receive KubeReader not kubeclient.Interface because:
//   - The interface is narrow: GetSecret and GetConfigMap only — no write access
//   - Providers must not write Kubernetes resources (Orkestra owns cluster state)
//   - A narrow interface is easier to mock in provider tests
//   - The kubeclient package is internal; providers should not depend on it
package generic

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/pkg/kubeclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// kubeReaderAdapter implements orktypes.KubeReader by delegating to kubeclient.
type kubeReaderAdapter struct {
	kube kubeclient.Interface
}

// GetSecret reads a Secret's data by name and namespace.
// Returns the decoded data map (base64 already decoded by the API server).
// Uses ResourceVersion "0" to read from the watch cache — avoids etcd.
func (a *kubeReaderAdapter) GetSecret(ctx context.Context, namespace, name string) (map[string][]byte, error) {
	secret, err := a.kube.Clientset().CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{
		ResourceVersion: "0", // watch cache
	})
	if err != nil {
		return nil, fmt.Errorf("getting secret %s/%s: %w", namespace, name, err)
	}
	return secret.Data, nil
}

// GetConfigMap reads a ConfigMap's data by name and namespace.
// Uses ResourceVersion "0" to read from the watch cache.
func (a *kubeReaderAdapter) GetConfigMap(ctx context.Context, namespace, name string) (map[string]string, error) {
	cm, err := a.kube.Clientset().CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{
		ResourceVersion: "0", // watch cache
	})
	if err != nil {
		return nil, fmt.Errorf("getting configmap %s/%s: %w", namespace, name, err)
	}
	return cm.Data, nil
}
