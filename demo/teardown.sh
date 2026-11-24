#!/usr/bin/env bash
# Полностью разбирает демо-окружение:
#   - удаляет kube-prometheus-stack и Envoy Gateway
#   - удаляет namespace deploy-demo
#   - (опционально) останавливает и удаляет minikube: ./teardown.sh --minikube
set -euo pipefail

WITH_MINIKUBE="${1:-}"

echo "Удаляю namespace deploy-demo ..."
kubectl delete namespace deploy-demo --ignore-not-found --wait=false || true

echo "Удаляю kube-prometheus-stack ..."
helm uninstall kps -n monitoring >/dev/null 2>&1 || true
kubectl delete namespace monitoring --ignore-not-found --wait=false || true

echo "Удаляю Envoy Gateway ..."
kubectl delete -f \
  "https://github.com/envoyproxy/gateway/releases/download/${ENVOY_GATEWAY_VERSION:-v1.10.0}/install.yaml" \
  --ignore-not-found >/dev/null 2>&1 || true

if [[ "$WITH_MINIKUBE" == "--minikube" ]]; then
  echo "Останавливаю и удаляю minikube ..."
  minikube delete || true
fi

echo "Готово."
