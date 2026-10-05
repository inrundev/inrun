// domain/generic.go
//
// Hook types for the Inrun reconcile framework.
//
// There are three layers here, each serving a different audience:
//
//  1. ReconcileHooks[T] — the user-facing API.
//     Users write: domain.ReconcileHooks[*Database]{OnReconcile: myFunc}
//     T is always a concrete pointer type (*Database, *Pipeline, etc.) because
//     Kubernetes informers store and return pointer values.
//
//  2. AnyReconcileHooks — the type-erased marker interface.
//     Allows the Catalog and ObjectRegistry to store hooks without knowing T.
//     The unexported isHooks() method prevents accidental implementation.
//
//  3. ObjectHooks / HookBinder — the internal adapter layer.
//     generic.Reconciler stores ObjectHooks, not ReconcileHooks[T], so that
//     a single reconciler type can serve both the typed user-hooks path
//     (T = *Database) and the dynamic template path (T = domain.Object)
//     that goes through the runtime registry in runtime_construct.go.
package domain

import "context"

// ReconcileHooks contains the user-provided business logic for a CRD.
// Only implement the hooks you need — all fields are optional.
//
// T must be a pointer to the concrete CR type (e.g. *Database).
// This matches how Kubernetes informers work: they store and return pointers,
// so type-asserting the informer cache entry to T succeeds only for *T, not T.
//
// Example:
//
//	func DatabaseHooks() domain.AnyReconcileHooks {
//	    return domain.ReconcileHooks[*apiv1.Database]{
//	        OnReconcile: onDatabaseReconcile,
//	        OnDelete:    onDatabaseDelete,
//	    }
//	}
type ReconcileHooks[T Object] struct {
	// OnReconcile is called for every create/update event.
	// obj is already type-asserted and deep-copied from the informer cache.
	// Return an error to requeue with backoff.
	OnReconcile func(ctx context.Context, obj T) error

	// OnDelete is called when the object's DeletionTimestamp is set.
	// Use this to clean up external resources before the finalizer is removed.
	// If nil, deletion proceeds with only finalizer removal.
	OnDelete func(ctx context.Context, obj T) error

	// OnNotFound is called when the object no longer exists in the store.
	// Use this for cleanup that depends on the key but not the object itself.
	// If nil, not-found events are a no-op.
	OnNotFound func(ctx context.Context, key string) error
}

// AnyReconcileHooks is the type-erased marker interface for ReconcileHooks[T].
// It allows the Catalog, ObjectRegistry, and HookFactory closures to store
// hooks without knowing the concrete T at compile time.
//
// The unexported isHooks() method prevents third-party types from accidentally
// satisfying this interface; only ReconcileHooks[T] values qualify.
type AnyReconcileHooks interface {
	isHooks() // unexported marker
}

// isHooks makes ReconcileHooks[T] satisfy AnyReconcileHooks.
func (r ReconcileHooks[T]) isHooks() {}

// ── Internal adapter layer ────────────────────────────────────────────────────
//
// generic.Reconciler stores ObjectHooks rather than ReconcileHooks[T] so it can
// hold a single concrete hooks value that works with both:
//
//   - The typed user path: T = *Database, hooks come from ReconcileHooks[*Database]
//   - The dynamic/template path: T = domain.Object (interface), hooks are nil
//
// The adapter is built once at construction time via HookBinder.BindToObjectHooks().
// Each wrapper closure performs a runtime obj.(T) assertion. This assertion is
// always safe because the informer only stores objects of the type it was
// constructed for — if the informer was built for *Database, every item in its
// cache IS a *Database even when retrieved as domain.Object.

// ObjectHooks is the type-erased counterpart to ReconcileHooks[T].
// generic.Reconciler stores this internally; users never construct it directly.
// Produce it by calling ReconcileHooks[T].BindToObjectHooks().
type ObjectHooks struct {
	OnReconcile func(ctx context.Context, obj Object) error
	OnDelete    func(ctx context.Context, obj Object) error
	OnNotFound  func(ctx context.Context, key string) error
}

// HookBinder is satisfied by every ReconcileHooks[T] value.
// generic.Reconciler calls BindToObjectHooks() through this interface at
// construction time so it can adapt any typed hooks to ObjectHooks without
// knowing the concrete T.
//
// Third-party hook wrappers that embed or delegate to ReconcileHooks[T] must
// also implement HookBinder to work with generic.New.
type HookBinder interface {
	AnyReconcileHooks
	// BindToObjectHooks wraps each hook function in a closure that performs
	// a runtime obj.(T) assertion before delegating to the typed function.
	// OnNotFound is forwarded unchanged — it receives a string key, not an object.
	BindToObjectHooks() ObjectHooks
}

// BindToObjectHooks implements HookBinder for ReconcileHooks[T].
// The returned ObjectHooks closures assert obj to T on every call.
// This is safe because the informer guarantees every object in its cache
// is of the type it was built for.
func (h ReconcileHooks[T]) BindToObjectHooks() ObjectHooks {
	oh := ObjectHooks{OnNotFound: h.OnNotFound}
	if h.OnReconcile != nil {
		fn := h.OnReconcile
		oh.OnReconcile = func(ctx context.Context, obj Object) error {
			return fn(ctx, obj.(T))
		}
	}
	if h.OnDelete != nil {
		fn := h.OnDelete
		oh.OnDelete = func(ctx context.Context, obj Object) error {
			return fn(ctx, obj.(T))
		}
	}
	return oh
}
