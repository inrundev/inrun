// Package merger resolves Catalog and Stack files, local or remote, into
// one set of CRD definitions for the runtime. A Catalog declares CRDs
// directly; a Stack imports Catalogs, OCI patterns, modules and Helm
// sources and merges them. Use New to build a Merger.
package merger
