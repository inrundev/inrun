// Package external runs a Katalog's declared external calls and puts their
// results in the template context under .external.<name>, both at reconcile
// time and during admission.
package external
