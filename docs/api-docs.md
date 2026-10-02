# Repair Desk — REST API

Base URL: `/api/v1` (via the frontend: `http://localhost:3000/api/v1`, directly: `http://localhost:8080/api/v1`).

- Protocol: HTTP/1.1, JSON (`Content-Type: application/json`).
- IDs are UUIDs, timestamps are RFC 3339 in UTC.
- Request bodies are strict: unknown fields are rejected with `400 INVALID_JSON`.
- Every response has an `X-Request-ID` header (taken from the request if present), which is also written to the logs.

## Contents
- [Error format](#error-format)
- [Health checks](#health-checks)
- [Categories](#categories)
- [Equipment](#equipment)
- [Tickets](#tickets)
- [Comments](#comments)
- [Business rules](#business-rules)

---

## Error format

Every error, including unknown routes and panics, has the same shape:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "request validation failed",
    "details": { "title": "is required" }
  }
}
```

`details` appears only for field validation errors.

| HTTP | `code` | When |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Invalid field, query parameter, or path UUID; a referenced category/equipment does not exist |
| 400 | `INVALID_JSON` | Malformed or empty body, unknown field, body larger than 1 MB |
| 400 | `EQUIPMENT_RETIRED` | A ticket is created for (or reassigned to) `RETIRED` equipment |
| 404 | `NOT_FOUND` | The entity with this ID does not exist |
| 404 | `ROUTE_NOT_FOUND` | Unknown URL |
| 405 | `METHOD_NOT_ALLOWED` | The route exists but not for this method (`Allow` header is set) |
| 409 | `ALREADY_EXISTS` | Duplicate category `name` or equipment `inventory_number` |
| 409 | `INVALID_STATUS_TRANSITION` | The status change is not allowed by the state machine |
| 409 | `TICKET_CLOSED` | Any change to a `CLOSED` ticket or its comments |
| 409 | `TICKET_NOT_DELETABLE` | Deleting a ticket that is not `NEW` |
| 409 | `CONCURRENT_UPDATE` | The ticket was changed by another request between read and write; reload and retry |
| 500 | `INTERNAL_ERROR` | Unexpected server error (details are only in the logs) |
| 503 | `NOT_READY` | `/readyz`: the database is unavailable |

---

## Health checks

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | Liveness: the process is running. `200 {"status":"ok"}` |
| GET | `/readyz` | Readiness: the database answers. `200 {"status":"ready"}` or `503 NOT_READY` |

---

## Categories

```json
{
  "id": "7bdfc566-1421-4354-9c42-f2d61f9d76c9",
  "name": "Network",
  "sla_hours": 4,
  "created_at": "2026-10-02T18:51:09.14796Z"
}
```

Seeded categories: Hardware (24 h), Software (48 h), Network (4 h), Printer (12 h), Other (72 h).

| Method | Path | Success | Description |
|---|---|---|---|
| GET | `/categories` | 200 `{"items": [Category]}` | Sorted by name |
| POST | `/categories` | 201 `Category` | Create |
| GET | `/categories/{id}` | 200 `Category` | |
| PUT | `/categories/{id}` | 200 `Category` | Replace `name` and `sla_hours` |
| DELETE | `/categories/{id}` | 204 | **Also deletes all tickets of this category** |

Request body (POST, PUT):

| Field | Type | Rules |
|---|---|---|
| `name` | string | required, ≤ 100 chars, unique |
| `sla_hours` | integer | 1 … 8760 |

Changing `sla_hours` does not change `due_at` of existing tickets.

---

## Equipment

```json
{
  "id": "07c11c49-ecd3-4212-b9b9-9d7352f42d3f",
  "name": "Dell Latitude",
  "inventory_number": "INV-001",
  "location": "Room 101",
  "status": "ACTIVE"
}
```

| Method | Path | Success | Description |
|---|---|---|---|
| GET | `/equipment?status=` | 200 `{"items": [Equipment]}` | Optional filter by `status` |
| POST | `/equipment` | 201 `Equipment` | Create |
| GET | `/equipment/{id}` | 200 `Equipment` | |
| PUT | `/equipment/{id}` | 200 `Equipment` | Replace all fields |
| DELETE | `/equipment/{id}` | 204 | Tickets stay, their `equipment_id` becomes `null` |

Request body (POST, PUT):

| Field | Type | Rules |
|---|---|---|
| `name` | string | required, ≤ 255 |
| `inventory_number` | string | required, ≤ 100, unique |
| `location` | string | required, ≤ 255 |
| `status` | string | `ACTIVE` (default) / `BROKEN` / `RETIRED` |

---

## Tickets

```json
{
  "id": "946ade9c-ef86-4c84-a22d-ba829ad953fe",
  "title": "No network",
  "description": "Cable is damaged",
  "status": "NEW",
  "priority": "HIGH",
  "category_id": "7bdfc566-1421-4354-9c42-f2d61f9d76c9",
  "equipment_id": "07c11c49-ecd3-4212-b9b9-9d7352f42d3f",
  "due_at": "2026-10-02T23:20:03.642678Z",
  "created_at": "2026-10-02T19:20:03.642678Z",
  "updated_at": "2026-10-02T19:20:03.642678Z",
  "overdue": false,
  "allowed_transitions": ["IN_PROGRESS", "CLOSED"]
}
```

`overdue` and `allowed_transitions` are computed on every read; the frontend uses them so it does not duplicate the business rules.

| Method | Path | Success | Description |
|---|---|---|---|
| GET | `/tickets` | 200 `TicketPage` | List with filters and pagination |
| POST | `/tickets` | 201 `Ticket` | Create |
| GET | `/tickets/{id}` | 200 `Ticket` | |
| PUT | `/tickets/{id}` | 200 `Ticket` | Edit fields (not `CLOSED`) |
| PATCH | `/tickets/{id}/status` | 200 `Ticket` | Change status |
| DELETE | `/tickets/{id}` | 204 | Only `NEW` tickets |

### GET /tickets

Query parameters (all optional, combined with AND):

| Parameter | Description |
|---|---|
| `status` | `NEW` / `IN_PROGRESS` / `RESOLVED` / `CLOSED` |
| `priority` | `LOW` / `MEDIUM` / `HIGH` / `CRITICAL` |
| `category_id` | UUID |
| `equipment_id` | UUID |
| `overdue` | `true` — only overdue tickets, `false` — only not overdue |
| `limit` | 1 … 200, default 50 |
| `offset` | ≥ 0, default 0 |

Response, sorted by `created_at` from newest:

```json
{ "items": [Ticket], "total": 37, "limit": 50, "offset": 0 }
```

### POST /tickets

```json
{
  "title": "Printer does not print",
  "description": "Paper jam on tray 2",
  "priority": "HIGH",
  "category_id": "…",
  "equipment_id": "…"
}
```

| Field | Rules |
|---|---|
| `title` | required, ≤ 255 |
| `description` | optional, ≤ 10000 |
| `priority` | optional, default `MEDIUM` |
| `category_id` | required, must exist (else `400 VALIDATION_ERROR`) |
| `equipment_id` | optional (`null` or omitted), must exist, must not be `RETIRED` (`400 EQUIPMENT_RETIRED`) |

The server sets `status = NEW` and `due_at = created_at + category.sla_hours`.

### PUT /tickets/{id}

```json
{ "title": "…", "description": "…", "priority": "LOW", "equipment_id": null }
```

All four fields are replaced. `category_id` and `status` cannot be changed here: the category is fixed at creation, the status changes only via `PATCH /status`. Assigning a different equipment that is `RETIRED` → `400 EQUIPMENT_RETIRED`; keeping equipment that was retired after linking is allowed. On a `CLOSED` ticket → `409 TICKET_CLOSED`.

### PATCH /tickets/{id}/status

```json
{ "status": "IN_PROGRESS" }
```

Allowed transitions (`docs/status.png`):

| From | To |
|---|---|
| `NEW` | `IN_PROGRESS`, `CLOSED` (cancel) |
| `IN_PROGRESS` | `RESOLVED` |
| `RESOLVED` | `IN_PROGRESS` (reopen), `CLOSED` |
| `CLOSED` | — |

Other transitions → `409 INVALID_STATUS_TRANSITION`; any change of a `CLOSED` ticket → `409 TICKET_CLOSED`.

### DELETE /tickets/{id}

Only for `NEW`, otherwise `409 TICKET_NOT_DELETABLE`. Comments are deleted together with the ticket.

---

## Comments

```json
{
  "id": "828e8514-900b-418c-a402-e07662be9f19",
  "ticket_id": "946ade9c-ef86-4c84-a22d-ba829ad953fe",
  "content": "Checked the cable",
  "created_at": "2026-10-02T19:20:04.512617Z"
}
```

| Method | Path | Success | Description |
|---|---|---|---|
| GET | `/tickets/{id}/comments` | 200 `{"items": [Comment]}` | Oldest first |
| POST | `/tickets/{id}/comments` | 201 `Comment` | `{"content": "…"}` |
| PUT | `/tickets/{id}/comments/{commentId}` | 200 `Comment` | `{"content": "…"}` |
| DELETE | `/tickets/{id}/comments/{commentId}` | 204 | |

`content`: required, ≤ 5000 chars. Any write on a comment of a `CLOSED` ticket → `409 TICKET_CLOSED`.

---

## Business rules

1. **SLA.** `due_at = created_at + category.sla_hours`, fixed at creation. Priority does not affect it.
2. **Status transitions** follow the table above; a ticket is always created as `NEW`.
3. **Retired equipment.** A ticket cannot be created for or reassigned to `RETIRED` equipment → `400`.
4. **Overdue.** A ticket is overdue if `due_at < now` and its status is not `RESOLVED` or `CLOSED`.
5. **Deletion.** Only `NEW` tickets can be deleted → otherwise `409`.
6. **Closed tickets are immutable**: no edits, status changes, or comment changes. `RESOLVED` tickets can be edited.
7. **Category is immutable** for a ticket after creation.
8. **Deleting a category** deletes its tickets; **deleting equipment** sets `equipment_id = null` in its tickets.

Rules that depend on the current status are enforced atomically in SQL (`UPDATE … WHERE status = $expected`), so concurrent requests cannot bypass them.

## Example

```sh
CAT=$(curl -s localhost:8080/api/v1/categories | jq -r '.items[] | select(.name=="Printer") | .id')
curl -s -X POST localhost:8080/api/v1/tickets \
  -H 'Content-Type: application/json' \
  -d "{\"title\":\"Paper jam\",\"category_id\":\"$CAT\"}"
```
