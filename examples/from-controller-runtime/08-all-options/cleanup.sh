#!/usr/bin/env bash
kubectl delete -f options/declarative/cr.yaml --ignore-not-found
kubectl delete -f options/hybrid/cr.yaml
kubectl delete -f options/hooks/cr.yaml 
kubectl delete -f options/constructor/cr.yaml
kubectl delete -f options/ork-resources/cr.yaml

kubectl delete -f options/declarative/crd.yaml
kubectl delete -f options/hybrid/crd.yaml
kubectl delete -f options/hooks/crd.yaml
kubectl delete -f options/constructor/crd.yaml
kubectl delete -f options/ork-resources/crd.yaml
