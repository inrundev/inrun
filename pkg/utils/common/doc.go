// Package common holds helpers shared by the runtime, gateway and katalog
// packages: health, metrics and annotation lookups on objects, and cross-CR
// HTTP fetches. It must stay a leaf and never import those packages.
package common
