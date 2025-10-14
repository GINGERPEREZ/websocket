# Prompt: Clean Architecture + Go + Kafka + WebSocket (Echo + Gorilla)

Usa este prompt como plantilla cuando necesites crear un nuevo módulo o servicio dentro de `realtime-go`. Pégalo en `README_ARCHITECTURE.md`, `shared/README.md` o donde prefieras.

---

## Plantilla: Clean Architecture + Go + Kafka + WebSocket (Echo + Gorilla)

Contexto:

- Proyecto: realtime-go
- Propósito: servicio encargado de consumir eventos desde Kafka y exponer actualizaciones via WebSocket.
- Principios: Clean Architecture (dependencias hacia adentro), DDD pragmático, inyección manual.

Estructura mínima recomendada:

```
realtime-go/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── <feature>/
│   │   ├── domain/
│   │   │   └── *.go       # Entidades, value objects
│   │   ├── application/
│   │   │   └── *.go       # Usecases + ports (interfaces)
│   │   ├── infrastructure/
│   │   │   └── *.go       # Implementaciones concretas (Kafka, Hub)
│   │   └── transport/
│   │       └── *.go       # Echo handlers / WebSocket adapter
│   └── shared/
│       └── utils, errors, logger
└── go.mod
```

Reglas clave:

1. Las dependencias apuntan hacia adentro. Domain no importa infra ni transport.
2. Define interfaces (ports) en la capa que los consume.
3. Todos los métodos públicos aceptan context.Context.
4. Inyección manual de dependencias en cmd/server/main.go.
5. Usa un Registry de TopicHandlers: cada handler expone Topic() y Handle(ctx, msg).

Ports y contratos imprescindibles:

- TopicHandler { Topic() string; Handle(ctx context.Context, msg \*domain.Message) error }
- Broadcaster { Broadcast(ctx context.Context, msg \*domain.Message) error }

Patrón recomendado:

- Kafka (producer) publica eventos desde REST.
- realtime-go actúa como Kafka consumer; usa Registry para inscribir handlers.
- Handlers construyen domain.Message y llaman a Broadcaster.
- Broadcaster puede ser HubBroadcaster (WebSocket hub) o cualquier adapter.

Ejemplo corto (referencia):

- domain/message.go : Message struct + NewMessage(...)
- application/pubsub_port.go : TopicHandler + Broadcaster interfaces
- infrastructure/websocket_hub.go : Hub, Client, HubBroadcaster
- infrastructure/registry.go : registry of TopicHandlers
- transport/http_handler.go : Echo WebSocket upgrade + register client to hub

Testing:

- Unit: domain and usecases with mocks.
- Integration: kafka consumer + registry; hub behavior.
- E2E: Echo + WebSocket handshake and message flow.

Notas operativas:

- Versiona eventos (user.created.v1).
- Maneja clientes lentos con bounded channels y drop + cleanup.
- Instrumenta métricas en el Hub (connections, drops).
- Mantén DTOs planos entre capas, no expongas entidades.

Scaffold prompt (IA / cli):
"Genera un módulo Go llamado <feature> que siga la estructura domain/application/infrastructure/transport con TopicHandler registry y un Hub WebSocket basado en Gorilla. Incluye tests unitarios mínimos y un wiring básico en cmd/server/main.go."

---

Recomendaciones finales:

- Prefiere registro declarativo de handlers (`registry.Register`) antes que switches.
- Mantén event types versionados (`user.created.v1`).
- Usa DTOs planos entre capas — no exportes entidades.
- Define errores claros en `shared/errors.go` con códigos (`ErrTopicNotFound`, `ErrInvalidPayload`).
- Implementa monitoring hooks en el Hub (para métricas, reconexiones).

---

Si quieres, puedo generar ahora el scaffold runnable completo en `internal/realtime` y actualizar `cmd/server/main.go` con wiring mínimo (creando los archivos mostrados y tests básicos). Dime y lo creo.
