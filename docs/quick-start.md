# Repair Desk + Grafana — короткая инструкция

Нужны Docker Desktop с Docker Compose и `make`. Запусти Docker Desktop и дождись готовности. Go, Node.js и PostgreSQL отдельно устанавливать не нужно.

## 1. Перейти в проект

```sh
cd /Users/gore/Desktop/SecondBrain/Personal/Projects/Repair-Desk
```

Все команды ниже выполняй из этой папки.

## 2. Создать настройки — только в первый раз

```sh
cp -n .env.example .env
```

Создаёт `.env`, не перезаписывая существующий. Для новой базы укажи свой `POSTGRES_PASSWORD` (латинские буквы и цифры). Если база уже создана, сохрани её текущие настройки.

## 3. Скачать образы и запустить всё

```sh
make local-pull
make local-up
```

Первая команда скачивает образы по одному сервису, вторая собирает и запускает приложение вместе с мониторингом. Порядок приложения: PostgreSQL → миграции → API → frontend. Дополнительно запускаются Grafana, VictoriaMetrics, экспортер PostgreSQL, пробер и алерты. Первый запуск может занять несколько минут; контейнеры работают в фоне.

Если скачивание завершилось с `EOF`, повтори `make local-pull`: уже скачанные слои сохраняются. После успешной загрузки выполни `make local-up`.

## 4. Проверить и открыть

```sh
make local-ps
curl -f http://localhost:8080/readyz
curl -f http://localhost:3001/api/health
```

У `postgres` и `api` ожидается `healthy`, у остальных сервисов — `Up`. `migrate` должен завершиться с `Exited (0)` — это успешное выполнение миграций. API возвращает `{"status":"ready"}`, Grafana — JSON с `"database": "ok"`.

- **Фронтенд:** http://localhost:3000 — категории, оборудование и заявки.
- **Grafana:** http://localhost:3001 — вход `admin` / `admin` при первом запуске с настройками по умолчанию. Далее Grafana предложит сменить пароль. В разделе Dashboards → Repair Desk доступны дашборды приложения и PostgreSQL. Источник VictoriaMetrics подключён автоматически; первые данные появляются примерно за 10–30 секунд.
- **Алерты:** http://localhost:9093 — Alertmanager; http://localhost:8095 — полученные уведомления.

Порты можно изменить в `.env`: `FRONTEND_PORT=3000`, `HTTP_PORT=8080`, `GRAFANA_PORT=3001`. Начальный пароль Grafana задаётся через `GRAFANA_ADMIN_PASSWORD` до первого запуска; существующий пароль хранится в её томе.

## 5. Остановить / запустить повторно

```sh
make local-down
```

Останавливает приложение и мониторинг, сохраняя базу, настройки Grafana и метрики. Для повторного запуска достаточно `make local-up`.

## Если возникла ошибка

```sh
make local-logs s=api
make local-logs s=grafana
```

Выполняй нужную команду: она показывает логи выбранного сервиса. Выход — `Ctrl+C`, контейнер продолжит работу. Для других сервисов меняй `s`, например на `postgres` или `frontend`.

Если Docker недоступен — проверь Docker Desktop. Если порт занят — измени его в `.env` и повтори `make local-up`. Kubernetes для этого варианта запуска не нужен.

Проверено 15.11.2026: сборка и запуск успешны; фронтенд `/` и `/tickets`, его JS/CSS и API отвечают HTTP 200; Grafana `/login` — HTTP 200, `/api/health` — `database: ok`. Оба дашборда загружены, запрос через источник данных Grafana возвращает `pg_up=1`; все три цели сбора метрик доступны, `repairdesk_probe_success=1`.
