#!/usr/bin/env bash
# Сбрасывает приложение между стратегиями: удаляет все Deployment/Service/HTTPRoute
# в namespace deploy-demo, НЕ трогая сам Gateway, мониторинг и namespace.
# Используйте перед переходом к следующей стратегии, чтобы селекторы не пересекались.
set -euo pipefail
NS="${NS:-deploy-demo}"

echo "Удаляю приложения и маршруты в namespace $NS ..."
kubectl delete httproute --all -n "$NS" --ignore-not-found
kubectl delete deployment my-app my-app-v1 my-app-v2 -n "$NS" --ignore-not-found
kubectl delete service my-app my-app-v1 my-app-v2 -n "$NS" --ignore-not-found
echo "Готово. Gateway 'eg', ServiceMonitor и monitoring на месте."
echo "Можно применять следующую стратегию."
