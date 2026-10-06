#!/usr/bin/env bash
set -euo pipefail

echo "Cleaning up intent..."

# The CR came from the gateway, not a file. Delete it while inrun is still
# running, so it can remove its finalizer.
kubectl delete website hello-intent --ignore-not-found
kubectl delete -f manifests/crd.yaml --ignore-not-found

echo "✓ Done. Stop 'inrun' and 'inrun gate run' with Ctrl+C if still running."
