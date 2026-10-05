// Package template evaluates Go template expressions against a live CR. A
// Resolver wraps the CR's object map and its Resolve method is the single
// evaluation surface for webhooks, gates, resource templates, status fields
// and autoscale conditions. Values without template syntax pass through.
package template
