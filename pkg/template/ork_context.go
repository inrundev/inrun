package template

import (
	"context"

	"github.com/orkspace/orkestra/pkg/utils"
)

type orkContextKey struct{}

// ContextWithOrkContext stores an OrkContext in a context.Context so that
// NewResolver can pick it up automatically without any call-site changes.
func ContextWithOrkContext(ctx context.Context, ork OrkContext) context.Context {
	return context.WithValue(ctx, orkContextKey{}, ork)
}

// orkContextFromCtx extracts the OrkContext from ctx, returning a zero value
// when none has been stored.
func orkContextFromCtx(ctx context.Context) OrkContext {
	if v, ok := ctx.Value(orkContextKey{}).(OrkContext); ok {
		return v
	}
	return OrkContext{}
}

// OrkContext holds runtime facts about the Orkestra process — constant for the
// lifetime of the process. Injected into every resolver as ".ork.*" so any
// template expression can read them without a note or a function call.
//
//	{{ .ork.inPod }}   → true when running inside a Kubernetes pod
//	{{ .ork.namespace }}   → the namespace Orkestra is deployed in
//	{{ .ork.version }}     → the running Orkestra version string
type OrkContext struct {
	InPod     bool   `json:"inPod"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

// NewOrkContext builds the runtime context. Called once at startup.
func NewOrkContext(namespace, version string) OrkContext {
	return OrkContext{
		InPod:     utils.IsRunningInPod(),
		Namespace: namespace,
		Version:   version,
	}
}

// asMap returns the context as a plain map for injection into the resolver data map.
func (o OrkContext) asMap() map[string]interface{} {
	return map[string]interface{}{
		"inPod":     o.InPod,
		"namespace": o.Namespace,
		"version":   o.Version,
	}
}
