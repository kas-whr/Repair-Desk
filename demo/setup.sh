#!/usr/bin/env bash
# Поднимает демо-окружение семинара «Стратегии деплоя» на локальном minikube:
#   1. minikube (docker-драйвер)
#   2. Envoy Gateway + Gateway API CRD
#   3. kube-prometheus-stack (Prometheus + Grafana)
#   4. namespace, Gateway, ServiceMonitor
# Приложение и конкретные стратегии применяются отдельно (см. README.md).
set -euo pipefail

ENVOY_GATEWAY_VERSION="${ENVOY_GATEWAY_VERSION:-v1.10.0}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { printf '\n\033[1;32m==> %s\033[0m\n' "$*"; }
die() { printf '\033[1;31mОШИБКА: %s\033[0m\n' "$*" >&2; exit 1; }

# --- 0. Зависимости -----------------------------------------------------------
for bin in kubectl helm minikube; do
  command -v "$bin" >/dev/null 2>&1 || die "не найден $bin — установите его и повторите"
done

# --- 1. minikube --------------------------------------------------------------
if minikube status >/dev/null 2>&1; then
  log "minikube уже запущен"
else
  log "Запускаем minikube (docker-драйвер, 4 CPU / 8 GB)"
  minikube start --driver=docker --cpus=4 --memory=8192 --disk-size=40g \
    --kubernetes-version=v1.32.0
fi
kubectl get nodes

# --- 2. Envoy Gateway + Gateway API ------------------------------------------
log "Устанавливаем Envoy Gateway ${ENVOY_GATEWAY_VERSION} (вместе с Gateway API CRD)"
kubectl apply --server-side -f \
  "https://github.com/envoyproxy/gateway/releases/download/${ENVOY_GATEWAY_VERSION}/install.yaml"
kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway \
  --for=condition=Available

# --- 3. kube-prometheus-stack -------------------------------------------------
log "Устанавливаем kube-prometheus-stack (Prometheus + Grafana)"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null
helm repo update >/dev/null
helm upgrade --install kps prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace \
  --set grafana.adminUser=admin \
  --set grafana.adminPassword=admin \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
  --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false \
  --wait --timeout 10m

# --- 4. База демо: namespace, Gateway, ServiceMonitor ------------------------
log "Применяем namespace, Gateway и ServiceMonitor"
kubectl apply -f "${HERE}/00-namespace.yaml"
kubectl apply -f "${HERE}/01-gateway.yaml"
kubectl apply -f "${HERE}/monitoring/servicemonitor.yaml"

# Дашборд Grafana — автопровиженинг: ConfigMap с меткой grafana_dashboard=1
# подхватывает sidecar kps-grafana, ручной импорт через UI не нужен.
log "Провиженим дашборд Grafana (K8s Deployment Strategies)"
kubectl create configmap grafana-dashboard-deploy-strategies \
  --namespace monitoring \
  --from-file=deploy-strategies.json="${HERE}/monitoring/grafana-dashboard.json" \
  --dry-run=client -o yaml \
  | kubectl label --local=true --dry-run=client -o yaml -f - grafana_dashboard="1" \
  | kubectl apply -f -

# На minikube LoadBalancer остаётся в Pending (нет внешнего IP), поэтому условие
# Programmed может не стать True без `minikube tunnel`. Ждём Accepted — его
# достаточно: Envoy-прокси поднимется, а доступ получим через port-forward (traffic.sh).
kubectl wait --timeout=3m -n deploy-demo gateway/eg --for=condition=Accepted

cat <<'EOF'

Готово. Дальше:
  # 1) трафик через Envoy-прокси (откроет порт и будет слать запросы)
  ./traffic.sh

  # 2) применить стратегию (пример: canary) и смотреть версии
  kubectl apply -f app/app-v1.yaml -f app/app-v2.yaml
  kubectl apply -f strategies/canary/httproute-90-10.yaml

  # 3) Grafana: пробросить порт (3001 — на 3000 может жить Grafana прошлого семинара)
  kubectl port-forward -n monitoring svc/kps-grafana 3001:80   # admin / admin
  # дашборд «K8s Deployment Strategies (Gateway API)» уже в Dashboards — провиженится автоматически

Полный пошаговый сценарий — в README.md.
EOF
