# Realtime WebSocket Module Guide

This guide explains how the realtime module is structured, how a WebSocket request flows through the system, and how to add new entities or capabilities while keeping the codebase aligned with Clean Architecture and DDD practices.

## Module Layout

```
internal/modules/realtime/
├── domain/           # Cross-entity message and pagination contracts
├── application/
│   ├── port/         # Inbound/outbound interfaces (Kafka, REST snapshots)
│   ├── usecase/      # Orchestration logic (connect, broadcast, caching)
│   └── handler/      # Kafka-driven application handlers
├── infrastructure/   # Hub, snapshot HTTP client, handler registry
└── interface/        # Echo WebSocket handler (transport layer)
```

Clean Architecture rules:

- **Domain** has no dependencies on application/infra/transport.
- **Application** depends only on domain and ports. It orchestrates flows and owns DTOs and cache policy.
- **Infrastructure** implements ports defined at the application layer (Kafka consumer, HTTP snapshot client, WebSocket hub).
- **Interface** adapts incoming protocols (Echo + Gorilla WebSocket) into application use cases.

## Request Lifecycle

1. **HTTP Upgrade** – `interface/http.handler.go` validates params and authorization, normalizes the entity, then upgrades the connection via Gorilla WebSocket.
2. **Use Case `ConnectSection`** – Validates the JWT (`shared/auth`) and initializes cache entries. No external calls are made until the client requests data or the broadcaster refreshes snapshots.
3. **Client Registration** – `infrastructure/websocket.hub.go` creates a `Client`, subscribes it to base topics (`{entity}.snapshot`, `{entity}.list`, `{entity}.detail`, `{entity}.error`, plus configured extras), and stores metadata for targeted broadcasts.
4. **Command Handling** – Incoming WebSocket messages trigger entity-specific command handlers in `interface/http.handler.go`, which delegate to `application/usecase/connect_section.go` for list/detail logic.
5. **Snapshot Fetching** – Use cases call `port.SectionSnapshotFetcher`, implemented by `infrastructure/section_snapshot_http_client.go`, to query the REST API. Responses are cached per section/entity/query.
6. **Broadcasting** – Domain changes arriving from Kafka are transformed into `domain.Message` structs by application handlers and relayed through `BroadcastUseCase` to the WebSocket hub.

## Analytics WebSockets

Analytics dashboards now have dedicated WebSockets under `/ws/analytics/:scope/:entity`.

- `scope` supports `public`, `restaurant`, and `admin`. Aliases such as `auth`, `rest`, `owner`, or singular nouns are normalized automatically.
- `entity` accepts REST analytics names (e.g. `users`, `dishes`, `menus`, `restaurants`, `sections`, `tables`, `images`, `objects`, `subscriptions`, `subscription-plans`, `reservations`, `reviews`, `payments`, `auth`).
- Authentication:
  - `public` scope works without a token, but accepts `Authorization: Bearer <token>` when available.
  - `restaurant` and `admin` scopes require a valid JWT in the `Authorization` header.
- Parameters:
  - Path identifiers are supplied as query parameters (e.g. `?restaurantId=...` for restaurant analytics).
  - Optional filters like `startDate`, `sectionId`, and `restaurantId` map directly to the REST analytics endpoints.
- Topics: clients automatically subscribe to `{analytics-<scope>-<entity>}.snapshot` and `{analytics-<scope>-<entity>}.error`.
- Commands: send `{"action":"refresh"}` or `{"action":"fetch","query":{"startDate":"2024-01-01"}}` to refresh data. Empty values remove filters.
- Actualizaciones: los dashboards se actualizan solos cuando llegan eventos Kafka relacionados (mesas, reservas, pagos, usuarios, etc.) gracias a la nueva caché de sesiones del use case. El servidor vuelve a consultar el REST de analytics y envía la respuesta únicamente a las sesiones afectadas.

Example connections:

```bash
# Public users analytics (no token required)
wscat -c ws://localhost:8080/ws/analytics/public/users

# Restaurant users analytics (token + restaurant queried via path param)
wscat -H "Authorization: Bearer <JWT>" \
   -c "ws://localhost:8080/ws/analytics/restaurant/users?restaurantId=rest-123"

# Admin payments analytics filtered by restaurant
wscat -H "Authorization: Bearer <ADMIN_JWT>" \
   -c "ws://localhost:8080/ws/analytics/admin/payments?restaurantId=rest-123&startDate=2024-01-01"
```

## Metadata & Payload Strategy

- `SectionSnapshot` always delivers the REST payload verbatim through the `Payload` field; metadata maps are optional and default to `nil`.
- We intentionally avoid deriving entity-specific metadata inside the Go adapter to keep the realtime layer decoupled from upstream REST schemas. Any enrichment must either come from the upstream service or be negotiated at the domain level.
- When future features require additional fields, prefer updating the REST response itself (so the payload remains source-of-truth) instead of reviving local normalizers. This prevents breakages whenever the REST contract evolves.
- Downstream websocket consumers should treat metadata maps as optional hints; feature-critical information must live in the payload to guarantee compatibility across services.

## Adding a New WebSocket Entity

Follow these steps to introduce an entity (e.g. `users`) that should expose realtime data:

1. **Domain Contracts**

   - Create command DTOs under `internal/modules/<entity>/domain` (e.g. `ListUsersCommand`, `GetUserCommand`).
   - Ensure domain types validate invariants (tests in the same package).

2. **REST Snapshot Support**

   - Extend `port.SectionSnapshotFetcher` with methods for list/detail retrieval if necessary.
   - Implement the new methods in `infrastructure/section_snapshot_http_client.go`, pointing to the appropriate REST endpoints.

3. **Use Case Logic**

   - In `connect_section.go`, add methods `List<Entity>` / `Get<Entity>` and corresponding `HandleList<Entity>Command` / `HandleGet<Entity>Command` wrappers that build `domain.Message` payloads.
   - Register list/detail cache refresh behavior if the broadcaster needs to rehydrate snapshots.

4. **Transport Handler**

   - Register a factory in `entityHandlers` (inside `interface/http.handler.go`) returning command handlers for the new entity. Respect alias actions (`list`, `fetch_all`, `detail`, `fetch_one`).
   - Normalize entity name aliases inside `normalizeEntity` to keep URLs friendly (`/ws/user/:section` → `users`).

5. **Kafka Integration**

   - Add the entity topics to configuration (`WS_ENTITY_TOPICS`, or defaults in `internal/config/config.go`).
   - Register Kafka handlers for the entity via `application/handler` and `infrastructure/registry.go`, ensuring broadcaster refreshes relevant cache entries.

6. **HTTP Wiring**

   - No additional routes are required; `/ws/:entity/:section` and `/ws/:entity/:section/:token` automatically route to the generic handler. Legacy aliases can be added when needed (`/ws/user/...`).
   - Update `WEBSOCKET_DEFAULT_ENTITY` if the fallback should point to the new entity.

7. **Documentation & Tests**
   - Add entity-specific payload examples to `docs/WEBSOCKET_ENDPOINTS.md`.
   - Cover new normalization paths in `interface/http.handler_test.go` and add unit tests for use cases.
   - If REST contracts change, update Swagger or API reference docs.

## Adding New Commands or Actions

1. Expand the entity-specific command handler factory to recognize the new `action` names.
2. Add corresponding methods in the use case (or extend existing ones) to perform the needed orchestration. Reuse `domain.BuildListMessage` / `domain.BuildDetailMessage` when possible.
3. Broadcast results using standardized topics. If the response deviates from list/detail, define new topic conventions (`{entity}.custom_action`).
4. Update `WEBSOCKET_ALLOWED_ACTIONS` (config) so clients subscribe to the new topics automatically.

## Configuration Cheatsheet

Environment variables consumed via `internal/config/config.go`:

- `WS_DEFAULT_ENTITY` – Fallback entity when clients use `/ws/section/:id` (defaults to `restaurants`).
- `WS_ALLOWED_ACTIONS` – Comma-separated list of action topics appended to base topics.
- `WS_ENTITY_TOPICS` – Mapping `entity:topicA|topicB` used to subscribe Kafka consumers.
- `REST_BASE_URL`, `REST_TIMEOUT` – Upstream REST service location for snapshots.
- `JWT_SECRET` – Required to validate tokens locally.

## Testing Strategy

- **Unit tests**: Ensure normalization, command handlers, and use cases return expected domain messages. Example: `internal/modules/realtime/interface/http.handler_test.go`.
- **Integration tests**: Mock REST responses or use a lightweight server to exercise `SectionSnapshotHTTPClient`.
- **Manual verification**: Use `wscat` or similar tools to connect:
  ```
  wscat -H "Authorization: Bearer <JWT>" -c ws://localhost:8080/ws/restaurants/main-hall
  ```
  Fire commands like `{"action":"list_restaurants"}` to verify responses.

## Operational Checklist

- [ ] Routes documented with sample curl/wscat commands.
- [ ] Kafka topics configured for the entity and registered in the broker registry.
- [ ] Use cases return `domain.Message` with proper metadata and timestamps.
- [ ] Snapshot cache invalidation tested for list/detail refresh flows.
- [ ] Token validation paths covered (missing/invalid/expired).
- [ ] Legacy routes updated or decommissioned if the entity replaces an old path.
- [ ] CI pipeline runs `go test ./...` and lints docs/code where available.

By following this guide, each new entity or WebSocket feature remains aligned with Clean Architecture, avoids duplication, and ships with clear documentation and tests.
