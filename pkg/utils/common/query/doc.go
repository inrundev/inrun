// Package query is the HTTP client other components use to read live state
// from the runtime. NewRuntimeQuery implements domain.RuntimeQuery; IsUnique
// is best effort, since it reads the informer cache.
package query
