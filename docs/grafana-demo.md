# Grafana в minikube: запуск и подготовка сдачи

Для задания используй этот вариант: Grafana должна работать в **minikube**. Docker Compose из [короткой инструкции](quick-start.md) подходит для локальной проверки, но сам по себе требование о minikube не закрывает.

В проекте уже подготовлены datasource и два дашборда. Grafana устанавливается манифестами (`make grafana-deploy`), что соответствует условию «любым удобным способом». Дополнительно устанавливать её через Helm не нужно.

## 1. Поднять стенд

Открой Docker Desktop. В терминале перейди в корень проекта:

```sh
cd /Users/gore/Desktop/SecondBrain/Personal/Projects/Repair-Desk
```

Если `minikube` или `helm` не установлены, на Mac с Homebrew:

```sh
brew install minikube helm
```

Если сейчас работает наш Compose-стенд, останови его, чтобы освободить ресурсы и порты. Данные сохранятся:

```sh
make local-down
```

Запусти кластер и выбери его для последующих команд:

```sh
minikube start --driver=docker --cpus=4 --memory=4096
kubectl config use-context minikube
```

Подготовь `.env`, если его ещё нет (`cp -n` сохраняет существующий файл). Затем собери образы внутри minikube и разверни приложение:

```sh
cp -n .env.example .env
make k8s-build
make k8s-deploy
kubectl -n repair-desk rollout status statefulset/postgres --timeout=300s
kubectl -n repair-desk wait --for=condition=complete job/migrate --timeout=600s
kubectl -n repair-desk rollout status deployment/api --timeout=300s
kubectl -n repair-desk rollout status deployment/frontend --timeout=300s
kubectl -n repair-desk rollout status deployment/prober --timeout=300s
```

`k8s-deploy` создаёт Secret БД из `.env`; миграции создают таблицы. Дождись успешного завершения каждой команды. Готовый Job миграции автоматически удаляется через 10 минут — это нормально.

Теперь VictoriaMetrics, затем Grafana:

```sh
make monitoring-deploy
kubectl -n monitoring rollout status deployment/vm-single-victoria-metrics-single-server --timeout=300s
```

Создай начальный пароль Grafana **один раз**, заменив пример своим значением. Не снимай пароль на скриншот и не включай его в видео:

```sh
kubectl -n monitoring create secret generic grafana-admin --from-literal=password='ReplaceWithYourPassword'
```

Если Secret уже существует, используй ранее заданный пароль. Затем:

```sh
make grafana-deploy
kubectl -n monitoring rollout status deployment/grafana --timeout=300s
kubectl -n repair-desk get pods
kubectl -n monitoring get pods
```

Поды приложения и Grafana должны быть `Running` и готовы (`READY` без нулей). У PostgreSQL ожидается `2/2`: БД и exporter. У API — два готовых пода. **Скриншот №00:** команды `kubectl config current-context` (ожидается `minikube`) и `get pods` в обоих namespace, подтверждающие запуск в Kubernetes.

Если rollout VictoriaMetrics не находит Deployment, выполни `kubectl -n monitoring get deployment,statefulset` и используй имя и тип установленного workload. При ошибке запуска смотри `kubectl -n monitoring describe pod <имя>` и `kubectl -n monitoring logs <имя>`.

## 2. Открыть порты — три отдельных терминала

В каждом терминале выполняй одну команду и оставляй её работать:

```sh
# Терминал 1: сайт
kubectl -n repair-desk port-forward svc/frontend 3000:80
```

```sh
# Терминал 2: Grafana
kubectl -n monitoring port-forward svc/grafana 3001:80
```

```sh
# Терминал 3: проверка VictoriaMetrics
kubectl -n monitoring port-forward svc/vm-single-victoria-metrics-single-server 8428:8428
```

Сайт: **http://localhost:3000**. Grafana: **http://localhost:3001**, логин `admin`, пароль из Secret. Сохраняем привычные порты: сайт — 3000, Grafana — 3001. Пример из задания с портом 3000 для Grafana использует другой внешний порт, это не обязательное условие.

Открой **http://localhost:8428/targets**: найди API, prober и postgres-exporter в namespace `repair-desk`. Их состояние должно стать `UP` после старта и нескольких циклов сбора. У API будут отдельные targets для двух подов.

## 3. Проверить datasource в Grafana

1. Слева открой **Connections → Data sources → VictoriaMetrics**.
2. Проверь тип **Prometheus** и URL **`http://vm-single-victoria-metrics-single-server:8428`**. Это адрес сервиса внутри namespace `monitoring`; `localhost:8428` здесь указывать не нужно.
3. Нажми **Save & test** / **Test**, если кнопка доступна. Источник управляется provisioning-файлом, поэтому поля могут быть недоступны для редактирования.
4. Открой **Explore**, выбери **VictoriaMetrics**, переключи запрос в режим **Code**, введи `pg_up` и нажми **Run query**. Ожидается ряд со значением `1`. Затем проверь `repairdesk_probe_success{namespace="repair-desk"}` — тоже `1` в норме.

Datasource добавляется автоматически из `k8s/monitoring/grafana/datasource.yaml`, его UID — `victoriametrics`. Если источника нет, повтори `make grafana-deploy` и проверь логи Grafana. Для ручного добавления путь: **Connections → Add new connection → Prometheus → Add new data source**, затем Name = `VictoriaMetrics` и указанный выше URL; для готовых JSON нужен UID `victoriametrics`, поэтому в этом проекте проще восстановить provisioning.

**Скриншот №01:** настройки VictoriaMetrics с URL и успешным тестом. Если тест недоступен — настройки и отдельный кадр Explore с успешным запросом.

## 4. Создать нагрузку, чтобы графики были живыми

В четвёртом терминале, из корня проекта:

```sh
python3 k8s/monitoring/load.py --url http://localhost:3000 --seconds 240 --workers 4
```

Это четыре минуты запросов чтения заявок через frontend → API → PostgreSQL, без удаления или изменения данных. Дождись хотя бы 1–2 минут накопления метрик. Для видео можно запустить нагрузку ещё раз.

В Grafana: **Dashboards → Repair Desk → Repair Desk — обзор и SLA**. Справа сверху выбери **Last 15 minutes**, автообновление **10s**. При необходимости нажми Refresh.

## 5. Что нажимать и скриншотить

Сохраняй кадры в `report7/screenshots/`. На Mac: `⌘⇧4` → выделить нужную область → переименовать PNG и переместить в папку. Оставляй видимыми названия панелей, шкалы, легенды и диапазон времени; мелкие графики лучше снять двумя кадрами.

1. **`00-minikube.png`** — подтверждение кластера и готовности подов из шага 1.
2. **`01-datasource.png`** — datasource VictoriaMetrics из шага 3. Дополнительный кадр Explore при необходимости назови `01b-explore.png`.
3. **`02-application.png`** — верх дашборда приложения: доступные API targets, результат пробера, RPS по HTTP-статусам, доля 5xx, p95 и доля быстрых запросов. В minikube в норме API targets = 2; нулевая доля ошибок — корректный результат.
4. **`03-prober-sla.png`** — прокрути до **«SLA пробера: хорошие минуты за 30 дней»** и **«Покрытие истории пробера за 30 дней»**; сними обе панели. Через меню `⋮` панели → **Inspect → Query** можно показать формулу; дополнительный кадр назови `03b-sla-query.png`.
5. **`04-runtime.png`** — нижняя часть: CPU, RSS, goroutines, heap, средняя пауза GC и длительность сценария пробера. Это детализация состояния Go-процесса. До первого GC соответствующая панель может не иметь данных; при таком результате явно укажи это в отчёте.
6. **`05-postgres.png`** — открой **Dashboards → Repair Desk → Postgres Overview**. Сверху выбери datasource `VictoriaMetrics`, доступные `job`, `instance` и БД `repairdesk` (либо своё имя из `.env`). Покажи QPS, Rows, Cache hit ratio, Number of active connections на фоне нагрузки. Оставь видимыми выбранную БД и интервал времени.

Для отдельной крупной панели: наведи курсор → `⋮` → **View**. Возврат на дашборд — кнопкой Back/стрелкой браузера.

**Про SLA:** 99% — порог SLA, 99.5% — внутренний SLO. Значение панели — оценка доступности по хорошим минутам за скользящие 30 дней, а не полный договорный SLA. На свежем стенде истории за месяц нет: обязательно покажи покрытие и напиши фактическую длительность наблюдений. Формула использует имеющиеся samples; пропуски внутри минуты сами по себе не доказывают успешность проверок. Для календарного месяца нужна отдельная оценка по границам месяца.

**Про PostgreSQL:** `Postgres Overview` — готовый дашборд №455, адаптированный к datasource проекта. Часть старых запросов Buffers может быть несовместима с PostgreSQL 17 / текущим exporter. Если там `No data`, проверь запрос в Explore и укажи ограничение; для демонстрации работы используй реальные данные QPS, Rows и Connections. Не выдавай отсутствие серии за нулевое значение.

## 6. Как показать именно импорт Postgres-дашборда

Provisioning уже загрузил оба JSON. Для наглядной демонстрации импорта через интерфейс:

1. **Dashboards → New → Import** (или **Import dashboard**).
2. Нажми **Upload dashboard JSON file** и выбери `k8s/monitoring/grafana/postgres.json`.
3. Укажи название **Postgres Overview — demo**, папку **Repair Desk** и **новый UID** `postgres-exporter-demo`, чтобы не перезаписывать provisioned-дашборд.
4. Если предлагается выбор datasource, выбери **VictoriaMetrics**. В подготовленном JSON его UID уже указан, поэтому отдельного выбора может не быть.
5. Нажми **Import**, выбери БД и интервал, покажи графики под нагрузкой. Можно использовать этот дашборд для скриншота №05.

Источник готового шаблона: [Postgres Overview, ID 455](https://grafana.com/grafana/dashboards/455-postgres-overview/). Импорт по ID тоже возможен, но JSON из проекта уже содержит привязку к нашему datasource.

## 7. Необязательная демонстрация ошибок

Для более показательного видео запусти нагрузку на 4 минуты, затем в отдельном терминале включи тестовую деградацию только учебного API:

```sh
kubectl -n repair-desk set env deployment/api FAULT_ERROR_RATE=0.3 FAULT_DELAY_MS=400
kubectl -n repair-desk rollout status deployment/api --timeout=300s
```

Подожди 1–2 минуты и покажи рост 5xx/p95, снижение доли быстрых запросов и возможные неудачи пробера. Скриншот при желании: `06-degradation.png`. После демонстрации обязательно восстанови нормальный режим:

```sh
kubectl -n repair-desk set env deployment/api FAULT_ERROR_RATE- FAULT_DELAY_MS-
kubectl -n repair-desk rollout status deployment/api --timeout=300s
```

Через несколько проверок последний результат пробера должен вернуться к `1`; накопленная оценка доступности не обязана сразу стать 100%.

## 8. Экспортировать JSON приложения

Открой **Repair Desk — обзор и SLA** → вверху **Export → Export as code**. В версиях с прежним меню: **Share → Export**. Выбери **Classic / JSON**, затем **Download file** / **Save to file**. Для повторного импорта в этот проект сохрани UID datasource `victoriametrics` (переключатель sharing with another instance можно оставить выключенным).

Сохрани скачанный файл как **`report7/application.json`**. Если менял панели, экспортируй именно окончательный вариант. Исходная версия лежит в `k8s/monitoring/grafana/application.json`, но для сдачи лучше приложить экспорт из демонстрируемой Grafana. CSV из Inspect → Data — другой артефакт, нужен именно JSON дашборда.

Проверка файла из корня проекта:

```sh
python3 -m json.tool report7/application.json >/dev/null
```

## 9. Сценарий скринкаста на 2–3 минуты

На Mac нажми `⌘⇧5` → **Запись выбранной области**; в параметрах выбери микрофон, если будешь комментировать. Запись останови кнопкой в строке меню.

1. **0:00–0:15** — терминал: контекст `minikube`, готовые поды приложения и Grafana в `monitoring`.
2. **0:15–0:35** — Grafana: datasource VictoriaMetrics, его URL, успешный тест или `pg_up=1` в Explore.
3. **0:35–1:10** — дашборд приложения под нагрузкой: статус, RPS, 5xx и p95; поясни RED: Rate, Errors, Duration.
4. **1:10–1:35** — отдельная панель SLA пробера + покрытие истории; затем CPU, RSS, heap и goroutines. Поясни, что ресурсные панели отражают utilization, но не измеряют все виды saturation.
5. **1:35–2:15** — импорт JSON Postgres с новым UID, выбор БД, живые QPS/Rows/Connections. Коротко объясни, что exporter собирает метрики PostgreSQL, VictoriaMetrics хранит их, а Grafana отображает.
6. **2:15–2:40** — экспорт JSON приложения и расположение артефактов. Деградацию можно показать дополнительным фрагментом, если уже подготовил данные.

Сохрани видео как `report7/demo.mov` или `report7/demo.mp4`. В репозитории `*.mp4` игнорируется: для сдачи приложи файл отдельно или дай ссылку с доступом и вставь её в отчёт.

## 10. Что сдавать

- `report7/application.json` — окончательный экспорт приложения.
- `report7/отчет.md` — заполненный [существующий отчёт](../report7/отчет.md) (при сдаче назови файл `Отчёт.md`), без оставшихся `[заполнить]`.
- `report7/screenshots/00-minikube.png` … `05-postgres.png` — шесть обязательных кадров; дополнительные — по необходимости.
- Короткое видео с обоими дашбордами и импортом Postgres.

Перед сдачей открой Markdown preview отчёта и проверь, что картинки отображаются, текст описывает именно наблюдавшиеся результаты, а ссылка на видео доступна проверяющему.

Команды minikube в этой инструкции подготовлены по манифестам проекта; выполнение задания в minikube ещё нужно подтвердить своими скриншотами. Ранее проверенный Compose-запуск не является подтверждением запуска в Kubernetes.

Справка по интерфейсу: [datasource Prometheus](https://grafana.com/docs/grafana/latest/datasources/prometheus/configure/), [импорт дашбордов](https://grafana.com/docs/grafana/latest/visualizations/dashboards/build-dashboards/import-dashboards/), [экспорт JSON](https://grafana.com/docs/grafana/latest/visualizations/dashboards/share-dashboards-panels/).
