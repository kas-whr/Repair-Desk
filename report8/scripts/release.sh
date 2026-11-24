#!/usr/bin/env bash
set -euo pipefail
export MSYS_NO_PATHCONV=1
cd "$(dirname "$0")/../.."
NS=repair-desk-release
K=(kubectl -n "$NS")
LOG_DIR="${LOG_DIR:-logs/releases}"

record() {
  mkdir -p "$LOG_DIR"
  printf '%s %s\n' "$(date -Iseconds)" "$*" | tee -a "$LOG_DIR/events.log"
}

route() {
  case "${1:-}" in
    blue|green|canary-90-10|canary-50-50|canary-0-100|ab|shadow|rolling) ;;
    *) echo 'STRATEGY must be blue, green, canary-90-10, canary-50-50, canary-0-100, ab, shadow or rolling' >&2; exit 1 ;;
  esac
  record "apply route $1"
  "${K[@]}" apply -f "k8s/releases/strategies/$1.yaml"
  # Check both conditions on the current generation, not stale status from the previous route.
  for ((i=0; i<60; i++)); do
    local generation conditions
    generation=$("${K[@]}" get httproute repairdesk-release -o jsonpath='{.metadata.generation}')
    conditions=$("${K[@]}" get httproute repairdesk-release -o jsonpath='{range .status.parents[*].conditions[*]}{.type}={.status}:{.observedGeneration}{"\n"}{end}')
    if grep -qx "Accepted=True:$generation" <<< "$conditions" && grep -qx "ResolvedRefs=True:$generation" <<< "$conditions"; then
      "${K[@]}" get httproute repairdesk-release -o yaml > "$LOG_DIR/route-$(date +%Y%m%d-%H%M%S)-$1.yaml"
      return
    fi
    sleep 1
  done
  "${K[@]}" describe httproute repairdesk-release
  exit 1
}

case "${1:-}" in
  gateway)
    kubectl apply --server-side -f "https://github.com/envoyproxy/gateway/releases/download/${ENVOY_GATEWAY_VERSION:-v1.10.0}/install.yaml"
    kubectl -n envoy-gateway-system wait deployment/envoy-gateway --for=condition=Available --timeout=300s
    kubectl apply -f k8s/releases/base/namespace.yaml
    kubectl apply -f k8s/releases/base/gateway.yaml
    "${K[@]}" wait gateway/repairdesk-eg --for=condition=Accepted --timeout=180s
    ;;
  secret)
    : "${POSTGRES_DB:?Set POSTGRES_DB in .env}" "${POSTGRES_USER:?Set POSTGRES_USER in .env}" "${POSTGRES_PASSWORD:?Set POSTGRES_PASSWORD in .env}"
    kubectl apply -f k8s/releases/base/namespace.yaml
    "${K[@]}" create secret generic postgres-credentials \
      --from-literal=POSTGRES_DB="$POSTGRES_DB" \
      --from-literal=POSTGRES_USER="$POSTGRES_USER" \
      --from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
      --dry-run=client -o yaml | kubectl apply -f -
    ;;
  build)
    eval "$(minikube docker-env --shell bash)"
    for version in v1.0.0 v2.0.0; do
      docker build -t "repair-desk-api:$version" --build-arg "VERSION=$version" .
    done
    ;;
  deploy)
    kubectl -n repair-desk rollout status statefulset/postgres --timeout=180s
    # The migration Job is removed by TTL after completion; verify persisted history too.
    if kubectl -n repair-desk get job/migrate >/dev/null 2>&1; then
      kubectl -n repair-desk wait job/migrate --for=condition=Complete --timeout=180s
    fi
    kubectl -n repair-desk exec statefulset/postgres -c postgres -- sh -c \
      'test "$(psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -tAc "SELECT count(*) FROM schema_migrations WHERE dirty = false")" = 1' \
      || { echo 'Apply database migrations first (make k8s-deploy).' >&2; exit 1; }
    kubectl apply -f k8s/releases/base/
    for deployment in release-v1 release-v2 release-shadow; do
      "${K[@]}" rollout status "deployment/$deployment" --timeout=240s
    done
    # Prepare the rolling workload too; only one route receives client traffic.
    "${K[@]}" apply -f k8s/releases/strategies/rolling-v1.0.0.yaml
    "${K[@]}" rollout status deployment/release-rolling --timeout=300s
    route blue
    ;;
  route) route "${STRATEGY:-blue}" ;;
  rolling)
    case "${RELEASE_VERSION:-v2.0.0}" in v1.0.0|v2.0.0) ;; *) echo 'RELEASE_VERSION must be v1.0.0 or v2.0.0' >&2; exit 1 ;; esac
    route rolling
    record "rolling ${RELEASE_VERSION:-v2.0.0}"
    "${K[@]}" apply -f "k8s/releases/strategies/rolling-${RELEASE_VERSION:-v2.0.0}.yaml"
    "${K[@]}" rollout status deployment/release-rolling --timeout=300s
    ;;
  reset)
    record 'reset to stable v1'
    route blue
    "${K[@]}" apply -f k8s/releases/strategies/rolling-v1.0.0.yaml
    "${K[@]}" rollout status deployment/release-rolling --timeout=300s
    ;;
  gateway-ui)
    service=$(kubectl -n envoy-gateway-system get svc \
      -l gateway.envoyproxy.io/owning-gateway-namespace="$NS",gateway.envoyproxy.io/owning-gateway-name=repairdesk-eg \
      -o jsonpath='{.items[0].metadata.name}')
    test -n "$service"
    kubectl -n envoy-gateway-system port-forward "svc/$service" "${GATEWAY_PORT:-8888}:80"
    ;;
  traffic)
    mkdir -p "$LOG_DIR"
    traffic() {
      local n=0 audience
      while true; do
        audience=default
        case "${AUDIENCE:-mixed}" in
          mixed) if (( n % 4 == 0 )); then audience=beta; fi ;;
          beta) audience=beta ;;
          default) ;;
          *) echo 'AUDIENCE must be mixed, default or beta' >&2; return 1 ;;
        esac
        printf '%s audience=%s ' "$(date -Iseconds)" "$audience"
        # Each curl opens a fresh connection. Keep errors in the evidence log and continue.
        curl --silent --show-error --max-time 3 \
          -H "x-user-group: $audience" -H "X-Demo-Run: ${DEMO_RUN:-manual}" \
          -w ' http_status=%{http_code}\n' "${GATEWAY_URL:-http://localhost:8888}/api/v1/version" || true
        n=$((n+1))
        sleep "${INTERVAL:-0.1}"
      done
    }
    traffic 2>&1 | tee -a "$LOG_DIR/traffic.log"
    ;;
  logs)
    mkdir -p "$LOG_DIR"
    stamp=$(date +%Y%m%d-%H%M%S)
    "${K[@]}" logs -l app=repairdesk-release --all-containers=true --prefix --timestamps \
      --max-log-requests=20 --since="${SINCE:-30m}" --tail=-1 > "$LOG_DIR/pods-$stamp.log"
    "${K[@]}" get pods -l app=repairdesk-release -o wide > "$LOG_DIR/pods-$stamp.txt"
    "${K[@]}" get httproute repairdesk-release -o yaml > "$LOG_DIR/route-$stamp.yaml"
    echo "Saved to $LOG_DIR (collect before rollback to retain terminating pod logs)."
    ;;
  *) echo 'Use gateway|secret|build|deploy|route|rolling|reset|gateway-ui|traffic|logs' >&2; exit 1 ;;
esac
