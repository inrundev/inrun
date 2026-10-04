// Package intent turns an intent into a custom resource and reads it back.
//
// An intent is the flat payload a caller sends to the gateway, in the
// caller's own vocabulary, with no apiVersion, kind or spec. Build maps it
// onto the CRD using the CRD's serve declaration. FromObject and Target
// read the intent and its target back from a stored resource, so the
// runtime can reconcile it per target.
package intent
