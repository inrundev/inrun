package kubeclient

import (
	"context"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// NewKubeclientFromConfig builds a fully started Kubeclient from an existing
// rest.Config. Used by the gateway cluster registry to construct clients for
// remote clusters declared in gateway.clusters. buildConfig short-circuits
// when restConfig is pre-seeded, so config is not required.
func NewKubeclientFromConfig(ctx context.Context, cfg *rest.Config, scheme *runtime.Scheme) (*Kubeclient, error) {
	k := &Kubeclient{
		name:       "remote",
		scheme:     scheme,
		started:    new(atomic.Bool),
		restConfig: cfg,
	}
	if err := k.Start(ctx); err != nil {
		return nil, err
	}
	return k, nil
}
