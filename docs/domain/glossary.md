# Mesa Ya Domain Glossary

This note captures the ubiquitous language we will lean on while refactoring the realtime service into a cleaner architecture. It does not introduce new behaviour; it only names the concepts already present in the REST contract and the websocket layer.

## Aggregates

- **RestaurantAggregate**
  - Root entity: `Restaurant`
  - Purpose: describe a venue that can receive reservations and owns layout sections.
  - Invariants: opening schedule must be valid (open time < close time); `totalCapacity` is the sum cap for hosted guests; subscription tier must exist.
  - Relations: owns zero or more `Section` aggregates; receives `Review` and `Reservation` references.
- **SectionAggregate**
  - Root entity: `Section`
  - Purpose: map a logical zone inside a restaurant.
  - Invariants: belongs to one restaurant; physical size (`width`, `height`) must be positive.
  - Relations: contains `Table` entities and layout `GraphicObject` instances.
- **TableAggregate**
  - Root entity: `Table`
  - Purpose: seat guests at a particular section position.
  - Invariants: unique number per section; capacity > 0; coordinates non-negative.
  - Relations: referenced by `Reservation`.
- **ReservationAggregate**
  - Root entity: `Reservation`
  - Purpose: capture a booking for a table at a specific date/time.
  - Invariants: number of guests <= table capacity; restaurant and table IDs must align.
- **ReviewAggregate**
  - Root entity: `Review`
  - Purpose: store feedback for a restaurant.
  - Invariants: rating between 1 and 5; reviewer is authenticated.
- **LayoutObjectAggregate**
  - Root entity: `GraphicObject`
  - Purpose: render auxiliary elements on a section plan (decorations, paths, etc.).
  - Invariants: coordinates and dimensions positive.
- **SectionObjectAggregate**
  - Root entity: `SectionObject`
  - Purpose: relate a `GraphicObject` with a `Section`.
  - Invariants: composite identity (sectionId, objectId) unique.

## Entities & Value Objects

- `RestaurantId`, `SectionId`, `TableId`, `ReservationId`, `ReviewId`, `UserId`: value objects wrapping UUIDs to avoid accidental mixups.
- `Schedule`: value object encapsulating `openTime`/`closeTime` with validation that spans a single day.
- `PagedQuery`: value object standardising `page`, `limit`, `search`, `sortBy`, `sortOrder` (already partially encoded in `SectionListOptions`).
- `GeoLocation`: free-form string today; candidate for future refinement.
- `Capacity`: positive integer; potential reuse across restaurants and tables.

## Domain Services

- **SnapshotService** (current `SectionSnapshotFetcher`): translates REST responses into domain aggregates for realtime consumption; today it returns raw maps—we will upgrade it to emit typed entities.
- **BroadcastService**: wraps Kafka/WebSocket broadcasting so use cases only emit domain messages.

## Context Map

- **Auth Context**: external provider for JWT validation, provides `Claims` consumed by the WebSocket handshake.
- **Backoffice REST Context**: source of truth for restaurant, section, table, reservation, and review data.
- **Realtime Context**: this service; subscribes to Kafka topics and exposes WebSocket commands.

## Immediate Refactor Targets

1. Promote `SectionListOptions` to a shared `PagedQuery` value object and reuse across restaurant and future section/table flows to keep DRY.
2. Convert `SectionSnapshotFetcher` to map REST payloads into `Restaurant` aggregates.
3. Align WebSocket command handlers to work with typed commands tied to aggregates, instead of generic JSON maps.

This glossary will evolve as we discover additional invariants or subdomains while moving code into the new structure.
