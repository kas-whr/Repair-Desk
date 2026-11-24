# Стратегии обновления Repair Desk: запуск и запись

Все команды выполняются из корня репозитория. На Windows используйте PowerShell
с установленными Git for Windows и GNU Make: Makefile сам запускает Bash-рецепты.
Нужны Docker Desktop, Go, minikube, kubectl, Helm; для minikube желательно 4 CPU / 8 GB RAM.
`demo/` — образец семинара. Его `setup.sh` запускать для этого сценария не нужно:
здесь используются собственное приложение и существующие VictoriaMetrics/Grafana.

## Что реализовано

- Данные хранятся в PostgreSQL с PVC в namespace `repair-desk`, все версии используют
  `postgres.repair-desk.svc.cluster.local`. В подах API постоянного состояния нет.
- У каждого release Deployment две реплики. `/healthz` проверяет процесс,
  `/readyz` проверяет доступность БД. Таймаут readiness больше таймаута Ping к БД.
- `GET /api/v1/version` возвращает версию, имя пода и сообщение; ответы содержат
  `X-App-Version`. Docker build встраивает версию через `ARG VERSION` и Go ldflags.
- Счётчик `repairdesk_release_http_requests_total{version,method,status,audience}`
  учитывает только `/api/`, без health/readiness. `audience` имеет значения default/beta.
  Метки `namespace`, `release`, `pod` добавляет сборщик VictoriaMetrics.
- Один `HTTPRoute repairdesk-release` меняется при выборе стратегии. Все сервисы
  подготовлены заранее, поэтому между опытами не нужно удалять namespace или БД.
- Shadow зеркалирует только GET в отдельный `release-shadow` с v2. Дополнительный
  middleware запрещает изменяющие запросы в shadow. Это защита приложения, а не
  отдельная роль PostgreSQL: для демонстрации используйте `/api/v1/version`.

## 1. Подготовка

Если `.env` ещё нет, скопируйте `.env.example` в `.env` и задайте параметры PostgreSQL.
У существующей БД они должны совпадать с учётными данными при её создании.
Не включайте секреты в запись экрана.

```powershell
# Если minikube уже работает, команду start пропустите.
minikube start --driver=docker --cpus=4 --memory=8192 --disk-size=40g
kubectl config use-context minikube

# Для первого запуска приложения/БД. Если они уже работают, эти две команды пропустите.
make k8s-build
make k8s-deploy
kubectl -n repair-desk rollout status statefulset/postgres --timeout=180s
if (kubectl -n repair-desk get job/migrate --ignore-not-found -o name) {
    kubectl -n repair-desk wait job/migrate --for=condition=Complete --timeout=180s
}

# Мониторинг (также обновляет настройки scrape существующей установки).
make monitoring-deploy
# Только если секрета Grafana ещё нет. Пароль ниже — для локального учебного стенда.
if (-not (kubectl -n monitoring get secret grafana-admin --ignore-not-found -o name)) {
    kubectl -n monitoring create secret generic grafana-admin --from-literal=password=local-demo-password
}
make grafana-deploy
kubectl -n monitoring rollout status deployment/grafana --timeout=180s

# Envoy Gateway, две версии приложения и начальный маршрут на v1.
make release-gateway
make release-build
make release-deploy
kubectl -n repair-desk-release get pods,svc,httproute
```

`release-deploy` проверяет БД и применённые миграции (Job может уже удалиться по TTL),
поднимает v1/v2/shadow/rolling и направляет трафик на v1. Его можно повторять;
это также вернёт Rolling на v1. Не выполняйте его посередине записи другой стратегии.

В каждой release Deployment должны быть `2/2` готовых реплики. У Gateway достаточно
`Accepted=True`: внешний IP LoadBalancer в minikube может оставаться Pending,
поскольку ниже используется port-forward. Для HTTPRoute команды проверяют
`Accepted=True` и `ResolvedRefs=True` именно для новой generation.

Теги v1.0.0/v2.0.0 обозначают фиксированные образы. Если вы изменили код и пересобрали
те же теги на уже работающем стенде, существующие поды продолжают использовать старый образ.
Перед записью один раз перезапустите их и дождитесь готовности:

```powershell
kubectl -n repair-desk-release rollout restart deployment/release-v1 deployment/release-v2 deployment/release-shadow deployment/release-rolling
kubectl -n repair-desk-release rollout status deployment/release-v1 --timeout=240s
kubectl -n repair-desk-release rollout status deployment/release-v2 --timeout=240s
kubectl -n repair-desk-release rollout status deployment/release-shadow --timeout=240s
kubectl -n repair-desk-release rollout status deployment/release-rolling --timeout=300s
```

## 2. Три постоянных терминала и Grafana

Терминал A — вход через **Envoy**, а не через Service приложения:

```powershell
make release-gateway-ui
```

Терминал B — непрерывный поток. В каждом цикле из четырёх запросов один beta,
остальные default. Используйте один генератор для всех опытов, не запускайте его копии.

```powershell
make release-traffic
```

Вывод содержит время, аудиторию, JSON с версией/подом и HTTP-статус. Он сохраняется
в `logs/releases/traffic.log`. Частота зависит от скорости машины; важно поддерживать
непрерывную нагрузку. Другие варианты: `make release-traffic AUDIENCE=default`
и `make release-traffic AUDIENCE=beta`. Остановить — Ctrl+C.

Терминал C — Grafana (3001, чтобы избежать конфликта с frontend на 3000):

```powershell
kubectl -n monitoring port-forward svc/grafana 3001:80
```

Откройте <http://localhost:3001/d/repairdesk-release-strategies>, войдите как `admin`
с паролем из `grafana-admin` (для созданного выше секрета — `local-demo-password`). Дашборд **Repair Desk - Release strategies**
провиженится `make grafana-deploy` из
[`release-dashboard.json`](../monitoring/grafana/release-dashboard.json).

Настройте дашборд:

1. Автообновление **5s**, диапазон **Last 15 minutes**.
2. **Audience = All** для сравнения обеих аудиторий.
3. **Workload = Rolling** для первого опыта; **Parallel** для остальных.
4. Основные панели: **Primary RPS by version** и **Primary response share**.
   **All backend traffic (including shadow)** отдельно показывает зеркальные копии.
   **Audience routing** доказывает A/B; **Traffic by pod** показывает смену реплик.

Панели используют `rate(...[1m])`, сбор метрик — каждые 15 секунд. После каждого
переключения выдерживайте **120 секунд** непрерывного трафика, чтобы получить
устойчивое плато. Рядом с моментом переключения линии сглажены минутным окном:
это не задержка самого HTTPRoute. Для доказательства быстроты Blue/Green используйте
также время команд и ответы генератора. Один график `rate()` не доказывает мгновенность.

Команды ниже выполняйте по шагам. Комментарий «120 секунд» означает паузу:
в PowerShell выполните `Start-Sleep -Seconds 120`, затем сделайте указанный
скриншот и только после этого переходите к следующей команде.

Запишите время начала каждого опыта из терминала/events.log. Для итогового скриншота
выберите абсолютный интервал от начала до конца именно этого опыта, включающий
исходный v1, переход и откат. Так предыдущая стратегия не попадёт в кадр.
В кадре оставляйте название дашборда, фильтры, время, легенду и шкалу процентов.

## 3. Rolling → скриншот → откат

Терминал D — команды управления. Сначала:

```powershell
make release-reset
make release-route STRATEGY=rolling
```

Выберите **Workload = Rolling**, дайте исходному v1 поработать 120 секунд.
Дополнительный терминал для видео и сохранения смены подов:

```powershell
kubectl -n repair-desk-release get pods -l release=release-rolling -L version -w | Tee-Object -FilePath logs/releases/rolling-pods.log
```

Обновление в терминале D:

```powershell
make release-rolling RELEASE_VERSION=v2.0.0
```

Команда ждёт завершения rollout. `maxSurge=1`, `maxUnavailable=0`,
`minReadySeconds=60`: Kubernetes последовательно заменяет две реплики, примерно
две минуты одновременно видны старые и новые поды. Точные доли не фиксированы:
они зависят от количества готовых endpoints и балансировки.

После завершения подождите 120 секунд. Сохраните `01-rolling-forward.png`:
v1 убывает, v2 растёт; на **Traffic by pod** видны разные версии одновременно.
Перед откатом сохраните логи v2:

```powershell
make release-logs
make release-rolling RELEASE_VERSION=v1.0.0
```

Дождитесь завершения и ещё 120 секунд. Сохраните `02-rolling-rollback.png`
с обоими переходами. Либо можно использовать `kubectl rollout undo`, но явное
применение v1 предсказуемо и при повторной записи. Затем:

```powershell
make release-logs
make release-reset
```

Остановите наблюдение подов через Ctrl+C. Для следующих опытов выберите **Workload = Parallel**.

## 4. Blue/Green → скриншот → откат

Здесь **blue = v1**, **green = v2** (названия цветов в demo могут отличаться).
Все поды уже готовы; меняется только HTTPRoute.

```powershell
make release-reset
# 120 секунд исходного v1
make release-route STRATEGY=green
# 120 секунд v2; сохраните 03-blue-green-forward.png
make release-logs
make release-route STRATEGY=blue
# 120 секунд v1; сохраните 04-blue-green-rollback.png
make release-logs
```

Ожидается v1 100% → v2 100% → v1 100%. Переключение маршрута не создаёт новые поды.
Переход применяется контроллером асинхронно, обычно за секунды, и не требует rollout.
На видео покажите изменение версии в генераторе сразу после команды и неизменные поды.
На минутном `rate()` переход визуально растянется примерно на минуту.

## 5. Canary 90/10 → 50/50 → 0/100 → откат

```powershell
make release-reset
# 120 секунд baseline
make release-route STRATEGY=canary-90-10
# 120 секунд; сохраните 05-canary-90-10.png
make release-logs
make release-route STRATEGY=canary-50-50
# 120 секунд; сохраните 06-canary-50-50.png
make release-logs
make release-route STRATEGY=canary-0-100
# 120 секунд; сохраните 07-canary-0-100.png (весь ramp в одном интервале)
make release-logs
make release-route STRATEGY=blue
# 120 секунд; сохраните 08-canary-rollback.png
make release-logs
```

На **Primary response share** доля v2 около 10%, затем около 50%, затем 100%,
после отката 0%. Это случайное распределение по весам, поэтому короткие интервалы
не обязаны точно совпадать с 90/10 или 50/50. При заметных колебаниях продлите этап
до 3–4 минут. На видео покажите также веса в сохранённом YAML маршрута.

## 6. A/B по аудитории → скриншот → откат

```powershell
make release-reset
# 120 секунд baseline
make release-route STRATEGY=ab
# 120 секунд смешанного трафика
curl.exe -s http://localhost:8888/api/v1/version
curl.exe -s -H "x-user-group: beta" http://localhost:8888/api/v1/version
make release-logs
```

Первый ответ — v1, второй — v2. На **Audience routing** при **Audience = All**
видны `default / v1.0.0` и `beta / v2.0.0`. Общая доля v2 около 25%, потому что
генератор отправляет 25% beta; это состав аудитории, а не случайный вес маршрута.
Сохраните `09-ab-audiences.png`. Дополнительно выберите **Audience = beta**:
на **Primary response share** для этой аудитории должно быть 100% v2.
Для **Audience = default** — 100% v1. Верните **All** перед откатом.

```powershell
make release-route STRATEGY=blue
# 120 секунд: обе аудитории получают v1
make release-logs
```

Сохраните `10-ab-rollback.png`: beta тоже вернулась на v1, хотя заголовки генератора
не менялись. Это отличает маршрутизацию по аудитории от Canary.

## 7. Shadow → скриншот → отключение зеркала

```powershell
make release-reset
# 120 секунд baseline
make release-route STRATEGY=shadow
# 120 секунд с зеркалированием
make release-logs
kubectl -n repair-desk-release logs -l release=release-shadow --prefix --tail=20
```

Сохраните `11-shadow-enabled.png`, включив в кадр следующие панели:

- **Primary response share**: v1 = 100%, пользовательские ответы v2 отсутствуют.
- **All backend traffic (including shadow)**: RPS `release-v1 / v1.0.0`
  примерно равен RPS `release-shadow / v2.0.0`.
- **Mirrored / stable request ratio**: около 100%. Генератор при этом видит только v1.

Здесь общее число обработок почти удваивается. Если просто сложить запросы обоих
backend и нормировать, получится примерно 50/50, но это **не доли ответов пользователям**.
Именно поэтому shadow исключён из панели primary share. Отдельный `release-v2`
не получает этот трафик; копии идут в `release-shadow`.

```powershell
make release-route STRATEGY=blue
# 120 секунд: shadow RPS и mirror ratio падают к 0, клиент всё ещё видит v1
make release-logs
```

Сохраните `12-shadow-disabled.png` с включением и отключением зеркалирования.
Держите **Workload = Parallel**, **Audience = All**.

## 8. Что приложить к заданию

Для каждой стратегии запишите видео: имя стратегии → исходный v1 → команда перехода
→ ответы генератора и Grafana → команда отката → возврат к v1. Для Rolling покажите
смену подов, для Canary — все три набора весов, для A/B — оба curl, для Shadow —
ответы v1 и логи/метрики shadow v2. Самих скриншотов недостаточно, если требуется видео.

Скриншоты можно сохранить в `report9/`, а логи взять из `logs/releases/`:
`events.log`, `traffic.log`, снимки HTTPRoute, подов и JSON-логов приложения.
`release-logs` вызывайте **до отката**: после удаления пода его логи kubectl уже
не прочитает. Для Rolling также сохраните вывод `-w` и видео: старые v1-поды
исчезают ещё во время обновления. Файлы в `logs/releases/` и `*.mp4` исключены из Git;
для сдачи приложите их отдельно или скопируйте нужные логи в каталог отчёта.

Для воспроизводимости дополнительно сохраните:

```powershell
kubectl version --client -o yaml > logs/releases/kubectl-version.yaml
minikube version > logs/releases/minikube-version.txt
kubectl -n envoy-gateway-system get deployment envoy-gateway -o yaml > logs/releases/envoy-controller.yaml
```

## Если Grafana показывает No data

1. Проверьте, что открыт дашборд Repair Desk, а источник — `VictoriaMetrics`
   с UID `victoriametrics`. `kps-grafana` из demo — другой стенд.
2. Проверьте терминалы port-forward и генератор, HTTP-статус должен быть 200.
3. Убедитесь, что фильтр **Workload** соответствует опыту и **Audience = All**.
   В запросах используется `${workload:raw}`: значение Parallel содержит регулярное
   выражение с `|`, которое нельзя экранировать как одно имя пода. Если вы открыли
   старую версию дашборда, выполните `make grafana-deploy` и обновите страницу.
4. Подождите минимум два scrape, затем полное минутное окно. В начале новая версия
   может отсутствовать в легенде, пока не получит первые запросы.
5. В Grafana Explore проверьте
   `up{namespace="repair-desk-release",app="repairdesk-release"}` (должно быть 1)
   и `repairdesk_release_http_requests_total{namespace="repair-desk-release"}`.
6. Если у счётчика нет метки `audience`, запущен старый образ: пересоберите и
   перезапустите release-поды, как описано в подготовке. Не делайте это во время записи.
7. Если scrape отсутствует, повторите `make monitoring-deploy`: здесь сбор метрик
   настроен через pod-аннотации в `k8s/monitoring/vmsingle-values.yaml`, ServiceMonitor не нужен.

Установка контроллера сверена с [официальной инструкцией Envoy Gateway](https://gateway.envoyproxy.io/docs/install/install-yaml/),
фильтр зеркалирования — с [поддержкой Gateway API в v1.10](https://gateway.envoyproxy.io/v1.10/concepts/gatewayapi-support/).
Поведение Rolling описано в [документации Kubernetes](https://kubernetes.io/docs/tasks/run-application/update-deployment-rolling/).
