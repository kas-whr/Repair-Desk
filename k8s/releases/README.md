# Release strategy demo

This is a small overlay for the existing Repair Desk. It keeps application state in PostgreSQL and runs two immutable image versions. Build v1 and v2 with `scripts/releases/apply.sh`, then apply one HTTPRoute at a time.

The current application version is returned by `GET /api/v1/version` and `X-App-Version`, and counted as `repairdesk_release_http_requests_total{version,...}`. Shadow is GET-only so mirrored requests cannot mutate the database.

Before deploying, create `postgres-credentials` in `repair-desk-release` or change the deployment env to the existing database secret. The manifests intentionally do not contain credentials.

Strategies:

- `rolling-v1.yaml` then `rolling-v2.yaml`: one Deployment, Kubernetes RollingUpdate.
- `blue.yaml` then `green.yaml`: switch one route between prepared Services.
- `canary-90-10.yaml`, `canary-50-50.yaml`, `canary-0-100.yaml`: change Gateway API backend weights.
- `ab.yaml`: `x-user-group: beta` goes to v2, everyone else to v1.
- `shadow.yaml`: user receives v1; GET traffic is mirrored to read-only v2.

Observe `repairdesk_release_http_requests_total` grouped by `version` in the existing VictoriaMetrics/Grafana setup. Capture traffic output, `kubectl get pods`, route YAML changes and metric graphs for the required video/logs.

Start the controller first with `make release-gateway`, then create the DB secret with `POSTGRES_DB=... POSTGRES_USER=... POSTGRES_PASSWORD=... make release-secret`. The existing repository's PostgreSQL service is not reused across namespaces automatically; point the release deployment at a database Service reachable from `repair-desk-release` if required.
