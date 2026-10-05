#!/usr/bin/env bash
set -euo pipefail

echo "Cleaning up remote..."

# Delete the CR while ork run is still running, so it can remove its finalizer.
kubectl delete -f manifests/cr.yaml --ignore-not-found
kubectl delete -f manifests/crd.yaml --ignore-not-found
kubectl delete -f manifests/secret.yaml --ignore-not-found
kubectl delete -f reconciler/deploy.yaml --ignore-not-found

echo "✓ Done. Stop 'ork run' and the reconciler with Ctrl+C if still running."
