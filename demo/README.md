# Демо-окружение: стратегии деплоя на Gateway API + minikube

Локальное окружение к семинару «Kubernetes. Стратегии деплоя». Всё поднимается на
вашей машине: **minikube** → **Envoy Gateway** (реализация Gateway API) →
**kube-prometheus-stack** (Prometheus + Grafana) → тестовое приложение
`containersol/k8s-deployment-strategies` в двух версиях.

Показываем три стратегии вживую — **Rolling**, **Blue/Green**, **Canary** — и две
справочно (**A/B**, **Shadow**). Трафик между версиями дробит **Gateway API**
(`HTTPRoute`), а не Ingress и не Istio VirtualService.

> Терминология: здесь и в слайдах везде «демо-окружение» / «окружение».

---

## Что нужно

- Docker (запущен)
- `minikube`, `kubectl`, `helm` в `PATH`
- ~4 CPU и ~8 GB RAM для minikube, ~40 GB диска
- Доступ в интернет (тянем образы и чарты)

Проверить: `minikube version && kubectl version --client && helm version --short`.

---

## Быстрый старт

```bash
cd 2026-2027/seminars/deploy-strategies/demo

# 1. Поднять окружение: minikube + Envoy Gateway + Prometheus/Grafana + база демо
./setup.sh

# 2. В отдельном терминале — генератор трафика через Envoy-прокси
./traffic.sh
```

`traffic.sh` сам найдёт Envoy-прокси, пробросит локальный порт `:8888` и будет
печатать версию, которая ответила:

```text
21:04:11 | Host: my-app-v1-6d4f7, Version: v1.0.0
```

`setup.sh` применяет только базу (namespace, Gateway, ServiceMonitor). Приложение
и маршруты каждой стратегии применяются отдельно — см. ниже.

> **Правило: одна стратегия за раз.** Все стратегии используют namespace
> `deploy-demo` и один Gateway `eg`. Перед переходом к следующей стратегии
> выполните `./reset.sh` — он удалит приложения и маршруты, оставив Gateway и
> мониторинг на месте.

---

## Доступ к Grafana

```bash
kubectl port-forward -n monitoring svc/kps-grafana 3001:80   # admin / admin
```

Откройте http://localhost:3001 → **Dashboards** → «K8s Deployment Strategies
(Gateway API)». Дашборд провиженится автоматически (`setup.sh` создаёт ConfigMap
с меткой `grafana_dashboard=1` — sidecar Grafana подхватывает его сам), ручной
импорт `monitoring/grafana-dashboard.json` не нужен.

> Порт именно **3001**: на `localhost:3000` обычно уже висит Grafana
> демо-окружения прошлого семинара «Логи и метрики» (docker-compose), и
> port-forward на 3000 либо не поднимется, либо вы попадёте не в ту Grafana.

Дашборд строит `sum(rate(http_requests_total{namespace="deploy-demo"}[2m])) by (version)`
— доли трафика между v1 и v2. Именно на нём видна разница стратегий.

---

## Стратегия 1 — Rolling (постепенный)

Один Deployment `my-app`, Kubernetes сам заменяет поды по одному. Gateway API не
дробит трафик — `HTTPRoute` всегда смотрит на сервис `my-app`.

```bash
# стартовая версия v1 (Deployment + Service + HTTPRoute)
kubectl apply -f strategies/rolling/01-app-v1.yaml
kubectl rollout status deploy/my-app -n deploy-demo

# в другом терминале уже крутится ./traffic.sh — видно только v1.0.0
# накатываем v2:
kubectl apply -f strategies/rolling/02-app-v2.yaml

# наблюдаем постепенную замену подов
kubectl get pods -n deploy-demo -l app=my-app -w
```

В `traffic.sh` версии v1 и v2 будут чередоваться, пока все поды не станут v2.
На графике — плавный спад v1 и рост v2. `maxSurge: 1` / `maxUnavailable: 0`
означают обновление без простоя.

Откат: `kubectl rollout undo deploy/my-app -n deploy-demo`.

```bash
./reset.sh   # перед следующей стратегией
```

---

## Стратегия 2 — Blue/Green

Обе версии подняты одновременно, трафик переключается одним движением.

```bash
# green (v1) и blue (v2)
kubectl apply -f app/app-v1.yaml -f app/app-v2.yaml
kubectl rollout status deploy/my-app-v1 -n deploy-demo
kubectl rollout status deploy/my-app-v2 -n deploy-demo

# весь трафик на green (v1)
kubectl apply -f strategies/blue-green/httproute-green.yaml
# traffic.sh показывает только v1.0.0

# МГНОВЕННОЕ переключение на blue (v2)
kubectl apply -f strategies/blue-green/httproute-blue.yaml
# traffic.sh тут же показывает только v2.0.0

# мгновенный откат — снова green
kubectl apply -f strategies/blue-green/httproute-green.yaml
```

На графике — прямоугольная ступенька: доля v2 скачком 0 → 100%.

```bash
./reset.sh
```

---

## Стратегия 3 — Canary (по весу)

Постепенно увеличиваем долю новой версии. Обе версии уже подняты.

```bash
kubectl apply -f app/app-v1.yaml -f app/app-v2.yaml
kubectl rollout status deploy/my-app-v1 -n deploy-demo
kubectl rollout status deploy/my-app-v2 -n deploy-demo

# 90% v1 / 10% v2
kubectl apply -f strategies/canary/httproute-90-10.yaml
# смотрим traffic.sh и график: v2 ~10%

# 50 / 50
kubectl apply -f strategies/canary/httproute-50-50.yaml

# 0 / 100 — полный перекат на v2
kubectl apply -f strategies/canary/httproute-0-100.yaml
```

Откат на любом шаге — применить файл с меньшим весом v2. На графике — плавная
рампа доли v2: 10% → 50% → 100%.

```bash
./reset.sh
```

---

## Справочно: A/B testing (по заголовку)

Не «случайные 10%», а точно выбранная аудитория. Здесь — по HTTP-заголовку.

```bash
kubectl apply -f app/app-v1.yaml -f app/app-v2.yaml
kubectl apply -f strategies/ab/httproute-ab.yaml

GW=http://localhost:8888   # порт уже проброшен traffic.sh
curl -s -H 'Host: my-app.local' "$GW/"                          # -> v1.0.0
curl -s -H 'Host: my-app.local' -H 'x-user-group: beta' "$GW/"  # -> v2.0.0
```

Заголовок `x-user-group: beta` уводит на v2, все остальные — на v1. Так же можно
матчить query-параметры, метод, cookie.

```bash
./reset.sh
```

---

## Справочно: Shadow (зеркалирование)

Пользователь получает ответ с v1, а копия запроса уходит на v2. Новая версия
проверяется боевым трафиком, не влияя на пользователей.

```bash
kubectl apply -f app/app-v1.yaml -f app/app-v2.yaml
kubectl apply -f strategies/shadow/httproute-shadow.yaml

# traffic.sh по-прежнему показывает только v1.0.0 (ответы идут с v1),
# но метрики и логи v2 растут — туда зеркалируется копия каждого запроса:
kubectl logs -n deploy-demo -l version=v2.0.0 --tail=20
```

> `RequestMirror` проверен вживую на Envoy Gateway v1.10 со стандартными CRD —
> работает без experimental-канала. Если у вас более старая версия Gateway API и
> фильтр не принимается, поставьте experimental-канал CRD:
> ```bash
> kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.7.0/experimental-install.yaml
> ```

```bash
./reset.sh
```

---

## Структура

```text
setup.sh            minikube + Envoy Gateway + kube-prometheus-stack + база демо
teardown.sh         разобрать всё (./teardown.sh --minikube — ещё и удалить minikube)
reset.sh            сбросить приложение между стратегиями (Gateway/мониторинг целы)
traffic.sh          генератор трафика через Envoy-прокси + port-forward
00-namespace.yaml   namespace deploy-demo
01-gateway.yaml     GatewayClass `eg` + Gateway `eg` (HTTP :80)
app/
  app-v1.yaml       Deployment+Service my-app-v1 (green / stable)
  app-v2.yaml       Deployment+Service my-app-v2 (blue / canary)
strategies/
  rolling/          01-app-v1.yaml, 02-app-v2.yaml (один Deployment my-app)
  blue-green/       httproute-green.yaml, httproute-blue.yaml
  canary/           httproute-90-10.yaml, -50-50.yaml, -0-100.yaml
  ab/               httproute-ab.yaml        (справочно)
  shadow/           httproute-shadow.yaml    (справочно)
monitoring/
  servicemonitor.yaml      ServiceMonitor для Prometheus Operator
  grafana-dashboard.json   дашборд «RPS по версиям»
```

---

## Как это устроено (коротко)

- **GatewayClass `eg`** говорит: «этот класс обслуживает контроллер Envoy Gateway».
- **Gateway `eg`** — точка входа (слушатель HTTP :80). Envoy Gateway поднимает под
  него Envoy-прокси в namespace `envoy-gateway-system`.
- **HTTPRoute `my-app`** — правила маршрутизации: на какие сервисы и в какой доле
  (`weight`) слать трафик. Меняя только этот ресурс, мы реализуем blue/green,
  canary, A/B и shadow — приложение не трогаем.
- **ServiceMonitor** объясняет Prometheus, где снимать метрики; метрика
  `http_requests_total{version=...}` приходит из самого приложения.

Полный разбор — в слайдах семинара: `../index.html`.

---

## Разбор окружения

```bash
./teardown.sh              # удалить Prometheus/Grafana, Envoy Gateway, namespace
./teardown.sh --minikube   # ...и остановить/удалить minikube
```
