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
- Metadata maps remain available but are optional; we now avoid deriving entity-specific fields inside the Go adapter to reduce coupling with REST payload shapes.
- Preserve backward-compatible keys (`tablesAvailable`, `reservationsConfirmed`, etc.) so downstream consumers do not need updates.
- Document adjustments alongside future steps; full contract refresh for `WEBSOCKET_TOPICS.md` happens after snapshot client normalizers land.

## Step 3 — Realtime domain refactor ✅

- Replaced typed fields in `section_snapshot.go` with metadata containers and added merge helpers.
- Rewrote `section_messages.go` to stitch metadata from the new containers; removed table/reservation summary helpers and trimmed imports.
- Simplified `section_snapshot_http_client.go` to forward normalized payloads without injecting derived metadata, keeping the transport layer generic.
- Refreshed `section_messages_test.go` to build snapshots via metadata maps; all realtime domain tests running again (`go test ./internal/modules/realtime/domain/...`).
- Next: tackle Step 4 (application layer) to drop the remaining typed command structs and make the HTTP handler fully generic.

## Step 4 — Application layer + handler refactor ✅

- `connect_section.go` now handles list/detail commands through the generic entity entry points and caches responses without typed DTOs.
- `SectionSnapshotFetcher` exposes only generic list/detail methods; the HTTP client implements the slimmer interface.
- The websocket handler maps every entity to the same generic command executor, eliminating legacy restaurant/table/reservation branches.
- Metadata normalization helpers were reduced to no-ops so the realtime module no longer depends on REST-specific fields.
- Remaining work: smoke the full websocket flow end-to-end (manual or automated) and confirm downstream consumers are comfortable without derived metadata.
