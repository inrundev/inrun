// Package merger resolves Katalog and Komposer files, local or remote, into
// one set of CRD definitions for the runtime. A Katalog declares CRDs
// directly; a Komposer imports Katalogs, OCI patterns, motifs and Helm
// sources and merges them. Use New to build a Merger.
package merger
