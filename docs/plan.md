## Step 1 — Typed usage audit ✅

Files that still import the legacy domain packages and why they matter:

- `internal/modules/realtime/domain/section_snapshot.go`: stores typed projections (`RestaurantList`, `TableList`, `ReservationList`, etc.) next to the raw payload so higher layers can reuse parsed data without re-decoding JSON.
- `internal/modules/realtime/domain/section_messages.go` (+ tests): enriches metadata for list/detail broadcasts using enums/fields from those typed structs (counts, schedule windows, reservation statuses, table states).
- `internal/modules/realtime/infrastructure/section_snapshot_http_client.go`: after decoding the REST payload, calls `Build*` helpers from each domain to populate the typed projections stored in the snapshot.
- `internal/modules/realtime/application/usecase/connect_section.go`: still expects the concrete command DTOs (`ListRestaurantsCommand`, `GetTableCommand`, etc.) and exposes handler wrappers around them.
- `internal/modules/realtime/interface/http.handler.go`: websocket command handlers decode payloads into the typed structs and route them to the use case. Removing the packages today breaks these imports immediately.

## Step 2 — Generic snapshot payload design (in progress)

Working proposal to make all entities behave the same way:

- Keep `SectionSnapshot.Payload` as `map[string]any` (already in place) and add lightweight helpers inside the realtime module to derive list/detail metadata purely from that map. No typed pointers from other modules.
- Introduce two new optional carriers inside `SectionSnapshot`:
  - `ListMeta *ListMetadata` with generic counters (items, total, custom map of aggregate metrics).
  - `DetailMeta map[string]string` for key-value facts we want to surface (status, schedule, etc.).
    These structures are populated by realtime-specific normalizers so we stay decoupled from restaurant/table/reservation enums.
- Define per-entity normalizer functions under `realtime/domain/normalizers` (one file per entity group). Each receives the raw payload (`map[string]any`) and returns `ListMetadata`/`DetailMeta` plus optional convenience slices (e.g. table state histogram). Future entities can plug a new normalizer without touching other modules.
- Update topic builders so they call these normalizers instead of reading typed fields. Metadata keys must remain backward compatible (`tablesAvailable`, `reservationsPending`, etc.); where necessary, the normalizer can translate enum strings coming from the REST payload.
- Ensure `SectionSnapshotHTTPClient` simply decodes JSON and invokes the appropriate normalizer via a registry keyed by entity, avoiding any dependency on domain-specific constructors. Unsupported entities fall back to a generic normalizer that only sets pagination info when possible.
- Keep command DTOs entirely inside the realtime module: define `ListCommandPayload` / `DetailCommandPayload` structs that mirror the JSON contract and use them for _all_ entities (current generic handlers already support this shape).
- Document the normalized metadata contract in `docs/websocket/WEBSOCKET_TOPICS.md` once the refactor lands so consumers know which keys remain stable.

Next step: detail the changes required in `section_messages.go` / `section_snapshot.go` to rely on the new metadata containers (Step 3).

## Step 3 — Realtime domain refactor (todo)

- **Redefine snapshot struct:** replace typed pointers in `internal/modules/realtime/domain/section_snapshot.go` with the new `ListMetadata` / `DetailMeta` containers and keep only the raw payload + generic helpers (e.g. `func (s *SectionSnapshot) ItemsCount() int`). Provide constructors to attach metadata produced by normalizers.
- **Introduce metadata types:** add `ListMetadata` (page, limit, itemsCount, totals, custom map) and helper methods to merge entity-specific aggregates without losing backward-compatible keys (`tablesAvailable`, `reservationsConfirmed`, etc.).
- **Plug normalizers:** create a registry in the domain package that, given an entity key, returns a function capable of extracting list/detail metadata from `map[string]any`. `section_snapshot_http_client.go` will call it later, but the definitions live here.
- **Refactor `section_messages.go`:** switch to the new metadata helpers instead of referencing restaurant/table/reservation structs. Keep emitted metadata identical; when not available, gracefully omit keys. Update detail/list builders to pull counts from `ListMetadata` and name/value facts from `DetailMeta`.
- **Update tests:** modify `section_messages_test.go` (and add new tests if needed) to build snapshots using the new helpers and verify that legacy metadata keys are still produced. Remove direct dependency on restaurant/table/reservation packages from the tests.
- **Regenerate coverage:** once changes compile, run `go test ./internal/modules/realtime/domain/...` to ensure metadata logic stays stable before moving to application layer adjustments in Step 4.
