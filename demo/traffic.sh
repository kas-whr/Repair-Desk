#!/usr/bin/env bash
# Шлёт непрерывный трафик через Envoy-прокси (Gateway API) на приложение и печатает
# версию, которая ответила. Так в реальном времени видно, как стратегия деплоя
# перераспределяет запросы между v1 и v2.
#
# Доступ к Envoy на minikube — через port-forward (LoadBalancer без `minikube tunnel`
# остаётся в Pending). Скрипт сам находит сервис прокси и пробрасывает локальный порт.
#
# Использование:
#   ./traffic.sh            # бесконечный цикл (Ctrl-C — стоп)
#   ./traffic.sh 100        # сделать 100 запросов и выйти
#   RATE=0.2 ./traffic.sh   # интервал между запросами, сек (по умолчанию 0.2)
set -euo pipefail

NS="${NS:-deploy-demo}"
EG_NS="${EG_NS:-envoy-gateway-system}"
LOCAL_PORT="${LOCAL_PORT:-8888}"
RATE="${RATE:-0.2}"
COUNT="${1:-0}"   # 0 = бесконечно

# --- находим Envoy-прокси, созданный для нашего Gateway `eg` -----------------
SVC="$(kubectl get svc -n "$EG_NS" \
  -l gateway.envoyproxy.io/owning-gateway-name=eg \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
if [[ -z "$SVC" ]]; then
  echo "Не найден сервис Envoy-прокси для Gateway 'eg' в namespace $EG_NS." >&2
  echo "Убедитесь, что setup.sh отработал и Gateway запрограммирован:" >&2
  echo "  kubectl get gateway -n $NS eg" >&2
  exit 1
fi

echo "Пробрасываю локальный порт :$LOCAL_PORT -> $EG_NS/svc/$SVC:80"
kubectl port-forward -n "$EG_NS" "svc/$SVC" "${LOCAL_PORT}:80" >/tmp/traffic-pf.log 2>&1 &
PF_PID=$!
trap 'echo; echo "Останавливаю port-forward (pid $PF_PID)"; kill "$PF_PID" 2>/dev/null || true' EXIT INT TERM

# ждём, пока порт откроется
for _ in $(seq 1 30); do
  curl -sf "http://localhost:${LOCAL_PORT}/" >/dev/null 2>&1 && break
  sleep 0.5
done

GW="http://localhost:${LOCAL_PORT}/"
echo "Трафик на $GW  (интервал ${RATE}s). Ctrl-C — остановить."
echo "-------------------------------------------------------------------"

i=0
while :; do
  body="$(curl -s "$GW" || echo '<нет ответа>')"
  printf '%s | %s\n' "$(date +%H:%M:%S)" "$body"
  i=$((i + 1))
  if [[ "$COUNT" -gt 0 && "$i" -ge "$COUNT" ]]; then break; fi
  sleep "$RATE"
done
