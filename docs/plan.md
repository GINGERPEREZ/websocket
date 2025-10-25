## Step 1 — Typed usage audit ✅

Original inventory (before refactor):

- `internal/modules/realtime/domain/section_snapshot.go`: now migrated to metadata containers (Step 3), removing the stored typed projections.
- `internal/modules/realtime/domain/section_messages.go` (+ tests): updated in Step 3 to consume metadata maps instead of enums/structs.
- `internal/modules/realtime/infrastructure/section_snapshot_http_client.go`: still uses legacy `Build*` helpers to derive metadata; will be revisited once we plug a normalizer registry.
- `internal/modules/realtime/application/usecase/connect_section.go`: still expects the concrete command DTOs (`ListRestaurantsCommand`, `GetTableCommand`, etc.) and exposes handler wrappers around them.
- `internal/modules/realtime/interface/http.handler.go`: websocket command handlers decode payloads into the typed structs and route them to the use case. Removing the packages today breaks these imports immediately.

## Step 2 — Generic snapshot payload design ✅

Decisions locked in for the implementation:

- `SectionSnapshot` keeps `Payload any` and exposes two generic maps (`ListMetadata`, `DetailMetadata`) instead of typed projections.
- Introduced `Metadata` (alias of `map[string]string`) plus merge helpers to attach entity aggregates without leaking whitespace-heavy values.
- Continue relying on entity-specific extractors for now (inside the HTTP client) but funnel every outcome through the metadata maps so higher layers stay agnostic of typed structs.
- Preserve backward-compatible keys (`tablesAvailable`, `reservationsConfirmed`, etc.) so downstream consumers do not need updates.
- Document adjustments alongside future steps; full contract refresh for `WEBSOCKET_TOPICS.md` happens after snapshot client normalizers land.

## Step 3 — Realtime domain refactor ✅

- Replaced typed fields in `section_snapshot.go` with metadata containers and added merge helpers.
- Rewrote `section_messages.go` to stitch metadata from the new containers; removed table/reservation summary helpers and trimmed imports.
- Updated `section_snapshot_http_client.go` to populate metadata maps using existing restaurant/table/reservation builders (temporary until dedicated normalizers replace them).
- Refreshed `section_messages_test.go` to build snapshots via metadata maps; all realtime domain tests running again (`go test ./internal/modules/realtime/domain/...`).
- Next: tackle Step 4 (application layer) to drop the remaining typed command structs and make the HTTP handler fully generic.
