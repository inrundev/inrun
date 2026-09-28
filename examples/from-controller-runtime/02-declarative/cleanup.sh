#!/usr/bin/env bash
kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
kubectl delete -f cr-with-secret.yaml --ignore-not-found
kubectl delete -f crd-with-secret.yaml --ignore-not-found
