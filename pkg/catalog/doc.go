// Package catalog holds a parsed Catalog: every CRD entry, enriched with
// what the runtime needs (API types from the CRD file, defaults, dependency
// order), and lookups by kind, name, GVK or serve target.
package catalog
