#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up constructor..."
kubectl delete -f manifests/cr.yaml --ignore-not-found
kubectl delete -f manifests/crd.yaml --ignore-not-found
echo "✓ Done. Stop 'ork run' with Ctrl+C if still running locally."
