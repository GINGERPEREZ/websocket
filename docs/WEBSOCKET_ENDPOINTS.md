# Websocket Endpoints

The realtime gateway exposes a single websocket entry point per entity. Every route requires a valid JWT issued by the auth service. Tokens can be provided as a route param, query string, or bearer header.

> Para una descripción completa del flujo interno y pasos para agregar nuevas entidades, consulta `docs/REALTIME_WEBSOCKET_GUIDE.md`.

## Routes

| Path                             | Description                                                                             |
| -------------------------------- | --------------------------------------------------------------------------------------- |
| `/ws/:entity/:section/:token`    | Primary route. Connects to the given `section` for the provided `entity`.               |
| `/ws/:entity/:section`           | Same as above but expects the token via `?token=` or an `Authorization: Bearer` header. |
| `/ws/restaurant/:section/:token` | Legacy alias that maps to the `restaurants` entity.                                     |
| `/ws/restaurant/:section`        | Legacy alias with token fallback.                                                       |

`entity` is case insensitive. Known values:

- `restaurants`
- `tables`
- `reservations`
- `reviews`
- `sections`
- `objects`
- `section-objects`
- `menus`
- `dishes`
- `images`
- `payments`
- `subscriptions`
- `subscription-plans`
- `auth-users`

Aliases such as `auth`, `auth_user`, `section_object`, or `subscriptionplans` are normalized automatically to the slugged forms above.

If the path segment resolves to `section`, `sections`, or `default`, the handler falls back to the configured default entity (`WEBSOCKET_DEFAULT_ENTITY`). Any other value must be integrated following the extension guide below.

## Connecting

```http
GET /ws/tables/main-hall/eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Header-only token (recommended)

```http
GET /ws/restaurants/main-hall HTTP/1.1
Host: localhost:8080
Upgrade: websocket
Connection: Upgrade
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

On a successful handshake the server emits a `system.connected` event containing the section, entity, and list of topics the client is subscribed to.

### Topics

The hub automatically subscribes each client to the base topics below (where `{entity}` is the normalized entity name):

- `{entity}.snapshot`
- `{entity}.list`
- `{entity}.detail`
- `{entity}.error`

Additional allowed actions from `WEBSOCKET_ALLOWED_ACTIONS` are appended as `{entity}.{action}`.

### Commands

Each entity exposes two command families:

- `list_*` (`list`, `fetch_all` aliases): responds with a `{entity}.list` message containing the latest snapshot.
- `get_*` (`detail`, `fetch_one` aliases): responds with `{entity}.detail` for the requested resource.

#### Restaurants

Payload contracts are defined in `internal/modules/restaurants/domain`. Example command:

```json
{
  "action": "list_restaurants",
  "payload": {
    "page": 1,
    "limit": 20,
    "search": "burger"
  }
}
```

#### Tables

```json
{
  "action": "get_table",
  "payload": {
    "id": "table-17"
  }
}
```

#### Reservations

#### Auth Users

```json
{
  "action": "get_auth_user",
  "payload": {
    "id": "user-42"
  }
}
```

```json
{
  "action": "list_reservations",
  "payload": {
    "page": 1,
    "limit": 10
  }
}
```

Errors are returned through `{entity}.error` messages containing the `reason` in metadata and data.

## Extending to a New Entity

Follow these steps to wire a new domain (for example `users`) into the websocket gateway:

1. **Model**: Create the domain module under `internal/modules/<entity>/domain` with the list/detail command DTOs.
2. **Ports**: Extend `internal/modules/realtime/application/port/section_snapshot_fetcher.go` only if a custom adapter is required; otherwise reuse the generic helpers.
3. **Infrastructure**: Update the `entityEndpoints` map in `internal/modules/realtime/infrastructure/section_snapshot_http_client.go` with the REST paths for list/detail operations.
4. **Use case**: Reuse `HandleListEntityCommand` / `HandleGetEntityCommand` or add specialized handlers if the payload diverges from the generic DTOs.
5. **Handler registry**: Register the entity in `entityHandlers` inside `internal/modules/realtime/interface/http.handler.go` (the generic handler covers most cases).
6. **Kafka**: Configure the new topics in `config/websocket.yaml` (or the corresponding environment variables) so the broadcaster receives change events.
7. **Docs**: Update this document with the new entity commands and any custom payload fields.
8. **Tests**: Add coverage for the new normalization branch and any command-specific behavior.

Once these steps are complete the `/ws/<entity>/<section>/<token>` route will accept connections and commands for the new entity.
