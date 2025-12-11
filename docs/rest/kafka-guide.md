# Guía completa de Kafka en MesaYa

Esta guía documenta al detalle cómo está integrada la mensajería con Apache Kafka en MesaYa. Aquí encontrarás los componentes disponibles, los pasos de configuración, ejemplos de uso de productores y consumidores, y buenas prácticas para mantener la integración al 10000000%.

## Visión general

La infraestructura Kafka vive en `src/shared/infrastructure/kafka` y se compone de:

- **KafkaModule**: módulo global de Nest que expone el servicio y explora los consumidores decorados.
- **KafkaService**: gateway centralizado que gestiona productor, servidores de consumidores y operaciones `emit`/`send`.
- **Decoradores**: utilidades para inyectar el servicio (`@KafkaProducer()`), registrar consumidores (`@KafkaConsumer()`) y emitir eventos declarativamente (`@KafkaEmit()`).
- **KafkaConsumerExplorer**: escanea los providers de Nest y registra automáticamente los métodos decorados como consumidores.
- **Tópicos normalizados**: definidos en `kafka.topics.ts` para mantener un naming consistente.

La meta es habilitar eventos de dominio desacoplados, reutilizables por cualquier feature sin romper la arquitectura limpia.

## Configuración requerida

Variables de entorno validadas al inicio (ver `shared/core/config/joi.validation.ts`):

| Variable          | Descripción                                         | Ejemplo             |
| ----------------- | --------------------------------------------------- | ------------------- |
| `KAFKA_BROKER`    | Lista de brokers separados por coma                 | `localhost:9092`    |
| `KAFKA_CLIENT_ID` | Identificador del cliente Kafka (opcional, default) | `mesa-ya`           |
| `KAFKA_GROUP_ID`  | Grupo consumidor por defecto                        | `mesa-ya-consumers` |

El módulo utiliza `ConfigService` para leer estos valores. Si `KAFKA_BROKER` o `KAFKA_GROUP_ID` no están configurados, la app falla durante el bootstrap para evitar estados inconsistentes.

## Componentes clave

### KafkaService

Ubicación: `shared/infrastructure/kafka/kafka.service.ts`

Funciones principales:

- Gestiona una única instancia de productor (`ClientKafka`), conectada bajo demanda.
- Administra servidores de consumidores (`ServerKafka`) agrupados por `groupId`.
- Expone `emit(topic, payload)` y `send(topic, payload)` (request/response).
- Permite registrar definiciones de consumidor en memoria y las inicializa durante `onApplicationBootstrap`.
- Serializa/limpia los payloads antes de enviarlos y asegura logs en caso de errores.

### Decoradores

- `@KafkaProducer()`: inyecta `KafkaService` vía DI manteniendo la capa de aplicación limpia.
- `@KafkaConsumer(topic, groupId?)`: añade metadatos para que el explorer registre el método como handler del tópico indicado.
- `@KafkaEmit({ topic, payload, includeTimestamp })`: envuelve métodos asincrónicos, ejecuta el caso de uso original y emite el evento correspondiente de forma automática. `payload` recibe `{ result, args, instance, toPlain }`.
- `toKafkaPlain`: helper reutilizable para forzar serialización a JSON plano.

### KafkaConsumerExplorer

- Se ejecuta en `onApplicationBootstrap`.
- Recorre los providers usando `DiscoveryService` y `MetadataScanner`.
- Por cada método decorado con `@KafkaConsumer` agrega la definición al `KafkaService`.
- Invoca `initializeConsumers()` para levantar los servidores necesarios.

### Kafka topics

Definidos en `kafka.topics.ts` y exportados vía `shared/infrastructure/kafka/index.ts`.

**DISEÑO OPTIMIZADO (Event-Driven Architecture):**

- Un tópico por dominio/agregado (en lugar de uno por acción)
- El tipo de evento se especifica en el payload con `event_type`
- Eventos efímeros (selecting/released) van por WebSocket directo, no Kafka
- Naming convention: `mesa-ya.{domain}.events`

```ts
// Event Types (usar en el campo `event_type` del payload)
export const EVENT_TYPES = {
  CREATED: 'created',
  UPDATED: 'updated',
  DELETED: 'deleted',
  STATUS_CHANGED: 'status_changed',
  USER_SIGNED_UP: 'user_signed_up',
  USER_LOGGED_IN: 'user_logged_in',
  ROLES_UPDATED: 'roles_updated',
  PERMISSIONS_UPDATED: 'permissions_updated',
} as const;

// Kafka Topics - Un tópico por dominio/agregado
export const KAFKA_TOPICS = {
  RESTAURANTS: 'mesa-ya.restaurants.events',
  SECTIONS: 'mesa-ya.sections.events',
  TABLES: 'mesa-ya.tables.events',
  OBJECTS: 'mesa-ya.objects.events',
  SECTION_OBJECTS: 'mesa-ya.section-objects.events',
  MENUS: 'mesa-ya.menus.events',        // includes dishes (entity_subtype: 'menu' | 'dish')
  REVIEWS: 'mesa-ya.reviews.events',
  IMAGES: 'mesa-ya.images.events',
  RESERVATIONS: 'mesa-ya.reservations.events',
  PAYMENTS: 'mesa-ya.payments.events',
  SUBSCRIPTIONS: 'mesa-ya.subscriptions.events',  // includes plans (entity_subtype)
  AUTH: 'mesa-ya.auth.events',
  OWNER_UPGRADE: 'mesa-ya.owner-upgrade.events',
} as const;

// Payload structure
interface KafkaEventPayload<T> {
  event_type: string;       // created | updated | deleted | status_changed | ...
  entity_id: string;
  entity_subtype?: string;  // For sub-entities (e.g., 'dish' within menus topic)
  timestamp: string;        // ISO 8601
  data: T;
  metadata?: {
    user_id?: string;
    correlation_id?: string;
  };
}
```

> Usa siempre estas constantes para evitar typos y garantizar estabilidad en los contratos.

## Cómo producir eventos

### Inyección con `@KafkaProducer()`

```ts
@Injectable()
export class ReviewsService {
  constructor(
    private readonly createReviewUseCase: CreateReviewUseCase,
    @KafkaProducer() private readonly kafkaService: KafkaService
  ) {}
}
```

### Emisión declarativa con `@KafkaEmit`

```ts
@KafkaEmit({
  topic: KAFKA_TOPICS.REVIEW_CREATED,
  payload: ({ args, result, toPlain }) => {
    const [command] = args as [CreateReviewCommand];
    return {
      action: 'review.created',
      entity: toPlain(result),
      performedBy: command.userId,
    };
  },
})
async create(command: CreateReviewCommand) {
  return this.createReviewUseCase.execute(command);
}
```

Ventajas:

- No repites boilerplate para `emit` ni `try/catch` en cada método.
- Puedes centralizar la estructura del mensaje en una sola función.
- El decorador añade automáticamente una marca de tiempo (`timestamp`) si no está presente.

### Alternativa manual

Si necesitas control total (casos muy específicos) puedes llamar directamente al servicio:

## Catálogo de eventos por servicio

**NUEVO DISEÑO EDA:** Todos los servicios ahora emiten al topic del dominio con `event_type` en el payload.

| Servicio                  | Método                  | Tópico                           | event_type           | Payload adicional                     |
| ------------------------- | ----------------------- | -------------------------------- | -------------------- | ------------------------------------- |
| `AuthService`             | `signup`                | `mesa-ya.auth.events`            | `user_signed_up`     | `entity_id`, `email`, `token`         |
| `AuthService`             | `login`                 | `mesa-ya.auth.events`            | `user_logged_in`     | `entity_id`, `email`, `token`         |
| `AuthService`             | `updateUserRoles`       | `mesa-ya.auth.events`            | `roles_updated`      | `entity_id`, `roles`                  |
| `AuthService`             | `updateRolePermissions` | `mesa-ya.auth.events`            | `permissions_updated`| `roleName`, `permissions`             |
| `ImagesService`           | `create`                | `mesa-ya.images.events`          | `created`            | `entity_id`, `data`                   |
| `ImagesService`           | `update`                | `mesa-ya.images.events`          | `updated`            | `entity_id`, `data`                   |
| `ImagesService`           | `delete`                | `mesa-ya.images.events`          | `deleted`            | `entity_id`                           |
| `PaymentService`          | `createPayment`         | `mesa-ya.payments.events`        | `created`            | `entity_id`, `data`                   |
| `PaymentService`          | `updatePaymentStatus`   | `mesa-ya.payments.events`        | `status_changed`     | `entity_id`, `status`                 |
| `PaymentService`          | `deletePayment`         | `mesa-ya.payments.events`        | `deleted`            | `entity_id`                           |
| `ReservationService`      | `create`                | `mesa-ya.reservations.events`    | `created`            | `entity_id`, `data`                   |
| `ReservationService`      | `update`                | `mesa-ya.reservations.events`    | `updated`            | `entity_id`, `data`                   |
| `ReservationService`      | `delete`                | `mesa-ya.reservations.events`    | `deleted`            | `entity_id`                           |
| `RestaurantsService`      | `create`                | `mesa-ya.restaurants.events`     | `created`            | `entity_id`, `data`                   |
| `RestaurantsService`      | `update`                | `mesa-ya.restaurants.events`     | `updated`            | `entity_id`, `data`                   |
| `RestaurantsService`      | `delete`                | `mesa-ya.restaurants.events`     | `deleted`            | `entity_id`                           |
| `ReviewsService`          | `create`                | `mesa-ya.reviews.events`         | `created`            | `entity_id`, `data`                   |
| `ReviewsService`          | `update`                | `mesa-ya.reviews.events`         | `updated`            | `entity_id`, `data`                   |
| `ReviewsService`          | `delete`                | `mesa-ya.reviews.events`         | `deleted`            | `entity_id`                           |
| `SectionObjectsService`   | `create`                | `mesa-ya.section-objects.events` | `created`            | `entity_id`, `data`                   |
| `SectionObjectsService`   | `update`                | `mesa-ya.section-objects.events` | `updated`            | `entity_id`, `data`                   |
| `SectionObjectsService`   | `delete`                | `mesa-ya.section-objects.events` | `deleted`            | `entity_id`                           |
| `SectionsService`         | `create`                | `mesa-ya.sections.events`        | `created`            | `entity_id`, `data`                   |
| `SectionsService`         | `update`                | `mesa-ya.sections.events`        | `updated`            | `entity_id`, `data`                   |
| `SectionsService`         | `delete`                | `mesa-ya.sections.events`        | `deleted`            | `entity_id`                           |
| `SubscriptionService`     | `create`                | `mesa-ya.subscriptions.events`   | `created`            | `entity_id`, `entity_subtype: 'subscription'` |
| `SubscriptionService`     | `update`                | `mesa-ya.subscriptions.events`   | `updated`            | `entity_id`, `entity_subtype: 'subscription'` |
| `SubscriptionService`     | `updateState`           | `mesa-ya.subscriptions.events`   | `status_changed`     | `entity_id`, `state`                  |
| `SubscriptionService`     | `delete`                | `mesa-ya.subscriptions.events`   | `deleted`            | `entity_id`, `entity_subtype: 'subscription'` |
| `SubscriptionPlanService` | `create`                | `mesa-ya.subscriptions.events`   | `created`            | `entity_id`, `entity_subtype: 'plan'` |
| `SubscriptionPlanService` | `update`                | `mesa-ya.subscriptions.events`   | `updated`            | `entity_id`, `entity_subtype: 'plan'` |
| `SubscriptionPlanService` | `delete`                | `mesa-ya.subscriptions.events`   | `deleted`            | `entity_id`, `entity_subtype: 'plan'` |
| `TablesService`           | `create`                | `mesa-ya.tables.events`          | `created`            | `entity_id`, `data`                   |
| `TablesService`           | `update`                | `mesa-ya.tables.events`          | `updated`            | `entity_id`, `data`                   |
| `TablesService`           | `delete`                | `mesa-ya.tables.events`          | `deleted`            | `entity_id`                           |
| `MenuService`             | `create`                | `mesa-ya.menus.events`           | `created`            | `entity_id`, `entity_subtype: 'menu'` |
| `MenuService`             | `update`                | `mesa-ya.menus.events`           | `updated`            | `entity_id`, `entity_subtype: 'menu'` |
| `MenuService`             | `delete`                | `mesa-ya.menus.events`           | `deleted`            | `entity_id`, `entity_subtype: 'menu'` |
| `DishService`             | `create`                | `mesa-ya.menus.events`           | `created`            | `entity_id`, `entity_subtype: 'dish'` |
| `DishService`             | `update`                | `mesa-ya.menus.events`           | `updated`            | `entity_id`, `entity_subtype: 'dish'` |
| `DishService`             | `delete`                | `mesa-ya.menus.events`           | `deleted`            | `entity_id`, `entity_subtype: 'dish'` |
| `ObjectsService`          | `create`                | `mesa-ya.objects.events`         | `created`            | `entity_id`, `data`                   |
| `ObjectsService`          | `update`                | `mesa-ya.objects.events`         | `updated`            | `entity_id`, `data`                   |
| `ObjectsService`          | `delete`                | `mesa-ya.objects.events`         | `deleted`            | `entity_id`                           |
| `OwnerUpgradeService`     | `updateStatus`          | `mesa-ya.owner-upgrade.events`   | `status_changed`     | `entity_id`, `status`                 |

```ts
// New pattern using @KafkaEmit decorator
@KafkaEmit({
  topic: KAFKA_TOPICS.REVIEWS,
  payload: ({ result, toPlain }) => ({
    event_type: EVENT_TYPES.CREATED,
    entity_id: result.id,
    data: toPlain(result),
  }),
})
async create(dto: CreateReviewDto) { ... }
```

Para mantener coherencia, limita esta modalidad a escenarios excepcionales.

## Cómo consumir eventos

1. Crea un provider o servicio y decora el método con `@KafkaConsumer`.

```ts
@Injectable()
export class ReviewEventsHandler {
  @KafkaConsumer(KAFKA_TOPICS.REVIEWS)
  async handleReviewEvent(payload: KafkaEventPayload, context: KafkaContext) {
    switch (payload.event_type) {
      case EVENT_TYPES.CREATED:
        // Handle creation
        break;
      case EVENT_TYPES.UPDATED:
        // Handle update
        break;
      case EVENT_TYPES.DELETED:
        // Handle deletion
        break;
    }
  }
}
```

2. Asegúrate de que el provider se registre dentro de un módulo importado por la aplicación (Nest lo detectará).

3. Opcional: especifica `groupId` para aislar consumidores. Si no lo haces, usa el `KAFKA_GROUP_ID` por defecto.

> Tip: utiliza DTOs o zod schemas para validar `payload` dentro del handler.

## Estrategia de payloads

- Cada evento incluye al menos: `action`, `entity` (para create/update/delete), `entityId` cuando aplica, `timestamp` y metadatos relevantes (`performedBy`, etc.).
- Las operaciones de eliminación publican la entidad completa tal y como existía antes de borrarla. Esto permite a los consumidores reconstruir el estado sin hacer llamadas adicionales.
- Usa `toPlain` para transformar entidades/DTOs en objetos simples sin prototipos de clase ni fechas crudas.
- Mantén backward compatibility: si introduces campos nuevos, hazlos opcionales y documenta el cambio.

```json
{
  "action": "restaurant.deleted",
  "entityId": "rest-123",
  "entity": {
    "id": "rest-123",
    "name": "MesaYa Centro",
    "ownerId": "owner-456",
    "status": "inactive"
  },
  "performedBy": "owner-456",
  "timestamp": "2025-10-12T23:59:59.000Z"
}
```

### Qué se emite en cada operación

| Operación | `action`                    | Cuerpo principal (`entity`)                                                     | Metadatos adicionales                                                         |
| --------- | --------------------------- | ------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| Create    | `mesa-ya.<feature>.created` | Entidad recién persistida (DTO de respuesta del caso de uso)                    | `performedBy` cuando la acción depende de un usuario (p. ej. ownerId, userId) |
| Update    | `mesa-ya.<feature>.updated` | Entidad resultante tras aplicar los cambios                                     | `performedBy`, `entityId` si el caso de uso recibe un identificador explícito |
| Delete    | `mesa-ya.<feature>.deleted` | Snapshot completo de la entidad antes de su eliminación (`entity` + `entityId`) | `performedBy` cuando aplica                                                   |

**Ejemplos completos (restaurant):**

```json
{
  "action": "restaurant.created",
  "entity": {
    "id": "rest-123",
    "name": "MesaYa Centro",
    "ownerId": "owner-456",
    "status": "active",
    "createdAt": "2025-10-12T10:00:00.000Z",
    "updatedAt": "2025-10-12T10:00:00.000Z"
  },
  "performedBy": "owner-456",
  "timestamp": "2025-10-12T10:00:00.000Z"
}
```

```json
{
  "action": "restaurant.updated",
  "entityId": "rest-123",
  "entity": {
    "id": "rest-123",
    "name": "MesaYa Centro",
    "ownerId": "owner-456",
    "status": "inactive",
    "createdAt": "2025-10-12T10:00:00.000Z",
    "updatedAt": "2025-10-13T09:30:15.000Z"
  },
  "performedBy": "owner-456",
  "timestamp": "2025-10-13T09:30:15.000Z"
}
```

```json
{
  "action": "restaurant.deleted",
  "entityId": "rest-123",
  "entity": {
    "id": "rest-123",
    "name": "MesaYa Centro",
    "ownerId": "owner-456",
    "status": "inactive",
    "createdAt": "2025-10-12T10:00:00.000Z",
    "updatedAt": "2025-10-13T09:30:15.000Z"
  },
  "performedBy": "owner-456",
  "timestamp": "2025-10-14T08:45:00.000Z"
}
```

**Secuencia interna del decorador `@KafkaEmit`:**

1. Ejecuta el método original (caso de uso / servicio) y captura el resultado (`result`).
2. Construye el payload con la función `payload` definida en el servicio. Ahí se recibe:
   - `result`: salida del método (los delete incluyen `{ ok, entity }`).
   - `args`: parámetros originales (útil para tomar IDs o usuarios ejecutores).
   - `toPlain`: helper que serializa DTOs y entidades a JSON plano.
3. Añade `timestamp` ISO8601 si el payload aún no lo trae.
4. Serializa el mensaje (internamente usa `JSON.parse(JSON.stringify(...))` para remover símbolos, fechas y prototypes).
5. Publica el evento vía `KafkaService.emit(topic, payload)` y registra cualquier error en el `Logger` del contexto sin interrumpir la respuesta HTTP.

> Gracias al snapshot incluido en `entity`, los consumidores pueden procesar eventos de eliminación sin consultas adicionales a la base de datos.

## Manejo de errores y resiliencia

- `KafkaEmit` atrapa cualquier error al construir o enviar el payload y los escribe en el `Logger` del contexto (o crea uno nuevo con el nombre de la clase).
- Si Kafka no está disponible, el endpoint HTTP sigue respondiendo con éxito para no bloquear la operación principal. Revisa los logs para detectar los fallos y aplica reintentos fuera del request (ej. workers o dead letter queues).
- El `KafkaService` limpia recursos durante `onModuleDestroy`, cerrando productor y consumidores correctamente.

## Testing

- **Unit tests**: mockea `KafkaService` para verificar que los payload builders retornan la estructura correcta. Puedes stubear el método `emit` y asegurarte de que se llame con el topic esperado.
- **Integration tests**: levanta un broker Kafka (por ejemplo usando `docker-compose`) y verifica que los mensajes llegan al topic correcto. Usa librerías como `kafkajs` para consumirlos dentro del test.
- **Contract tests**: define el esquema del evento (OpenAPI/AsyncAPI/JSON Schema) y valida que el payload emitido lo cumpla.

## Checklist para nuevas features

1. **Definir tópicos**: añade la constante en `kafka.topics.ts` siguiendo el patrón `mesa-ya.<feature>.<evento>`.
2. **Configurar permisos**: garantiza que el componente tenga acceso a `KafkaModule` (importa el módulo si aún no lo hace).
3. **Emitir eventos**: aplica `@KafkaEmit` en los métodos relevantes y documenta los campos del payload.
4. **Consumir eventos** (opcional): crea un handler con `@KafkaConsumer`, valida entrada y maneja errores.
5. **Documentar**: añade la descripción del evento y payload al README de la feature o al catálogo de eventos.
6. **Pruebas**: cubre casos exitosos y fallidos para los builders de payload y handlers.

## Diagnóstico y monitoreo

- `KafkaService` loguea cada conexión de productor y arranque de consumidor (`Kafka producer connected...`, `Kafka consumer ready...`).
- Ante errores de emisión o construcción de payload, revisa los logs con el contexto `<Clase>.error`.
- Considera añadir métricas (ej. Prometheus) envolviendo `KafkaService` para contar eventos enviados/errores.

## Referencias cruzadas

- **Código**: `src/shared/infrastructure/kafka/*`.
- **Uso en features**:
  - `reviews/application/services/reviews.service.ts`
  - `restaurants/application/services/restaurants.service.ts`
  - `sections/application/services/sections.service.ts`
- **Documentación complementaria**: sección "Integración con Kafka" en `docs/ARCHITECTURE.md`.

Con esta guía puedes incorporar o extender la mensajería Kafka de manera uniforme, evitando duplicar código y manteniendo los estándares de MesaYa.
