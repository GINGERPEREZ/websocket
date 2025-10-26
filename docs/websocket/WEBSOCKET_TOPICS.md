# Guía de tópicos WebSocket en MesaYa

Esta guía documenta al detalle cómo se publican y consumen los eventos del gateway WebSocket de MesaYa. Complementa a `docs/REALTIME_WEBSOCKET_GUIDE.md` (arquitectura interna) y al catálogo REST (`docs/swagger/swagger.yml`), ofreciendo una referencia unificada de entidades, comandos, tópicos y payloads.

## Visión general

- Ubicación del módulo: `internal/modules/realtime/*`.
- Entradas WebSocket: `GET /ws/:entity/:section/:token?` (token por parámetro, query o header `Authorization: Bearer`) y `GET /ws/analytics/:scope/:entity` (token solo por header; scope `public`, `restaurant`, `admin`).
- El hub suscribe automáticamente a todos los clientes a un conjunto de tópicos base y a los extra configurados vía `WS_ALLOWED_ACTIONS`.
- Las instantáneas provienen de la API REST documentada en Swagger, mientras que las actualizaciones en vivo llegan desde Kafka (ver `docs/kafka-guide.md`).

### Tópicos base

Para cualquier entidad registrada, el cliente queda suscrito a los tópicos:

| Tópico              | Uso                                                                            |
| ------------------- | ------------------------------------------------------------------------------ |
| `{entity}.snapshot` | Snapshot inicial emitido al conectarse o al refrescar la sección completa.     |
| `{entity}.list`     | Respuesta a comandos `list_*`. Incluye paginación normalizada y conteos.       |
| `{entity}.detail`   | Respuesta a comandos `get_*`. Contiene la entidad solicitada y metadatos.      |
| `{entity}.error`    | Errores de transporte/validación (payload inválido, recurso inexistente, etc). |

Extras opcionales (`{entity}.{action}`) se leen de `WS_ALLOWED_ACTIONS` (default: `created,updated,deleted,snapshot`). Estos tópicos reciben eventos generados por el broadcaster a partir de Kafka y comparten contratos con los tópicos homólogos descritos en `docs/kafka-guide.md`.

### Mensaje estándar

Todos los envíos reutilizan `domain.Message`:

```json
{
  "topic": "restaurants.list",
  "entity": "restaurants",
  "action": "list",
  "resourceId": "main-hall",
  "metadata": {
    "sectionId": "main-hall",
    "page": "1",
    "limit": "20",
    "search": "burger",
    "itemsCount": "5",
    "total": "27"
  },
  "data": {
    /* snapshot plano retornado por la API REST */
  },
  "timestamp": "2025-10-24T21:12:53.381Z"
}
```

Las llaves disponibles en `metadata` dependen del tipo de snapshot (listas, detalle, mesas, reservas) y se generan en `domain/section_messages.go`.

## Entidades integradas

| Entidad            | Alias aceptados                                                | Comandos soportados                                | REST (lista)                     | REST (detalle)                                        | Kafka (actualizaciones)¹                                                                   |
| ------------------ | -------------------------------------------------------------- | -------------------------------------------------- | -------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| restaurants        | `restaurant`, `restaurants`                                    | `list_restaurants`, `get_restaurant`               | `GET /api/v1/restaurant`         | `GET /api/v1/restaurant/{id}`                         | `mesa-ya.restaurants.{created,updated,deleted}`                                            |
| tables             | `table`, `tables`                                              | `list_tables`, `get_table`                         | `GET /api/v1/table`              | `GET /api/v1/table/{id}`                              | `mesa-ya.tables.{created,updated,deleted}`                                                 |
| reservations       | `reservation`, `reservations`                                  | `list_reservations`, `get_reservation`             | `GET /api/v1/reservations`       | `GET /api/v1/reservations/{id}`                       | `mesa-ya.reservations.{created,updated,deleted}`                                           |
| reviews            | `review`, `reviews`                                            | `list_reviews`, `get_review`                       | `GET /api/v1/review`             | `GET /api/v1/review/{id}`                             | `mesa-ya.reviews.{created,updated,deleted}`                                                |
| sections           | `section`, `sections`                                          | `list_sections`, `get_section`                     | `GET /api/v1/section`            | `GET /api/v1/section/{id}`                            | `mesa-ya.sections.{created,updated,deleted}`                                               |
| objects            | `object`, `objects`                                            | `list_objects`, `get_object`                       | `GET /api/v1/object`             | `GET /api/v1/object/{id}`                             | `mesa-ya.objects.{created,updated,deleted}`                                                |
| section-objects    | `section-object`, `section-objects`, `section_object`, etc.    | `list_section_objects`, `get_section_object`       | `GET /api/v1/section-object`     | `GET /api/v1/section-object/{id}`                     | `mesa-ya.section-objects.{created,updated,deleted}`                                        |
| menus              | `menu`, `menus`                                                | `list_menus`, `get_menu`                           | `GET /api/v1/menus`              | `GET /api/v1/menus/{menuId}`                          | `mesa-ya.menus.{created,updated,deleted}`                                                  |
| dishes             | `dish`, `dishes`                                               | `list_dishes`, `get_dish`                          | `GET /api/v1/dishes`             | `GET /api/v1/dishes/{dishId}`                         | `mesa-ya.dishes.{created,updated,deleted}`                                                 |
| images             | `image`, `images`                                              | `list_images`, `get_image`                         | `GET /api/v1/image`              | `GET /api/v1/image/{id}`                              | `mesa-ya.images.{created,updated,deleted}`                                                 |
| payments           | `payment`, `payments`                                          | `list_payments`, `get_payment`                     | `GET /api/v1/payments`           | `GET /api/v1/payments/{paymentId}`                    | `mesa-ya.payments.{created,updated,deleted}`                                               |
| subscriptions      | `subscription`, `subscriptions`                                | `list_subscriptions`, `get_subscription`           | `GET /api/v1/subscriptions`      | `GET /api/v1/subscriptions/{subscriptionId}`          | `mesa-ya.subscriptions.{created,updated,deleted}`                                          |
| subscription-plans | `subscription-plan`, `subscription-plans`, `subscription_plan` | `list_subscription_plans`, `get_subscription_plan` | `GET /api/v1/subscription-plans` | `GET /api/v1/subscription-plans/{subscriptionPlanId}` | `mesa-ya.subscription-plans.{created,updated,deleted}`                                     |
| auth-users         | `auth-user`, `auth-users`, `auth_user`, `auth_users`           | `list_auth_users`, `get_auth_user`                 | `GET /api/v1/auth/admin/users`²  | `GET /api/v1/auth/admin/users/{id}`²                  | `mesa-ya.auth.{user-signed-up,user-logged-in,user-roles-updated,role-permissions-updated}` |

¹Topicos Kafka definidos en `docs/kafka-guide.md` → `KAFKA_TOPICS`. El broadcaster escucha esos eventos y emite `{entity}.{created|updated|deleted}` a los clientes conectados en la sección correspondiente.

²Endpoints expuestos en el módulo de administración del servicio Auth; revisa `docs/swagger/swagger.yml` para confirmar permisos y contratos vigentes.

## Contrato de comandos

### 1. Comandos `list_*`

- Alias admitidos: `list_*`, `list`, `fetch_all`.
- Payload genérico (`domain.PagedQuery`):

| Campo       | Tipo   | Descripción                                         | Default tras normalizar |
| ----------- | ------ | --------------------------------------------------- | ----------------------- |
| `page`      | entero | Página solicitada (>=1).                            | `1`                     |
| `limit`     | entero | Tamaño de página (1-100).                           | `20` (máximo `100`)     |
| `search`    | string | Texto libre para filtrar.                           | `SectionID` si vacío    |
| `sortBy`    | string | Campo de ordenamiento (dependiente de la API REST). | `""`                    |
| `sortOrder` | string | `ASC` o `DESC` (case-insensitive).                  | `""`                    |

Ejemplo (`restaurants`):

```json
{
  "action": "list_restaurants",
  "payload": {
    "page": 2,
    "limit": 15,
    "search": "sushi",
    "sortBy": "name",
    "sortOrder": "asc"
  }
}
```

Respuesta → `{entity}.list` con metadata `page`, `limit`, `search`, `sortBy`, `sortOrder`, `itemsCount`, `total` (cuando aplica).

### 2. Comandos `get_*`

- Alias admitidos: `get_*`, `detail`, `fetch_one`.
- Payload específico por entidad:

| Entidad      | Payload mínimo               |
| ------------ | ---------------------------- |
| restaurants  | `{ "id": "rest-123" }`       |
| tables       | `{ "id": "table-17" }`       |
| reservations | `{ "id": "reservation-42" }` |

Ejemplo (`tables`):

```json
{
  "action": "get_table",
  "payload": {
    "id": "table-17"
  }
}
```

Respuesta → `{entity}.detail` con `metadata` enriquecido (estado de mesa, capacidad, resumen de reservas, etc.). Si el `id` está vacío, el servidor emite `{entity}.error` con `reason=invalid payload` sin consultar la API REST.

## Flujo de conexión

1. El transporte valida token y sección vía `ConnectSectionUseCase.Execute`. Errores frecuentes:
   - `missing token` → HTTP 400 (`usecase.ErrMissingToken`).
   - `invalid token` → HTTP 401 (`auth.ErrInvalidToken`).
   - `section not found` → HTTP 404 (`port.ErrSnapshotNotFound`).
2. Se crea un `Client` en `infrastructure/websocket.hub.go` y se registran tópicos mediante `buildTopics`.
3. Se envía `system.connected` con la lista de tópicos y roles.
4. El cliente puede emitir comandos inmediatamente; el servidor responderá en el mismo socket con `domain.Message` serializados a JSON.

## Ejemplos end-to-end

### Conexión y snapshot inicial

```bash
wscat -H "Authorization: Bearer <JWT>" -c ws://localhost:8080/ws/restaurants/main-hall
```

Mensajes esperados:

1. `system.connected` → confirmación con topics y roles.
2. `restaurants.snapshot` → snapshot inicial si la sección tiene datos cacheados.
3. Comandos manuales (`list_restaurants`, `get_restaurant`) según sea necesario.

### Actualizaciones provenientes de Kafka

1. Un caso de uso en Nest emite `mesa-ya.restaurants.updated` (ver decoradores `@KafkaEmit`).
2. El consumidor Kafka en Go (implementado en `internal/modules/realtime/application/handler`) procesa el evento, refresca el cache y ejecuta `BroadcastUseCase`.
3. El hub envía `restaurants.updated` y, si corresponde, actualiza snapshots (`restaurants.list` / `restaurants.detail`).

## Tabla de referencia rápida

| Acción cliente                        | Respuesta habitual                      | Fuente de datos                                 |
| ------------------------------------- | --------------------------------------- | ----------------------------------------------- |
| `list_restaurants`                    | `restaurants.list`                      | REST `GET /api/v1/restaurant`                   |
| `get_restaurant`                      | `restaurants.detail`                    | REST `GET /api/v1/restaurant/{id}`              |
| `list_tables`                         | `tables.list`                           | REST `GET /api/v1/table`                        |
| `get_table`                           | `tables.detail`                         | REST `GET /api/v1/table/{id}`                   |
| `list_reservations`                   | `reservations.list`                     | REST `GET /api/v1/reservations`                 |
| `get_reservation`                     | `reservations.detail`                   | REST `GET /api/v1/reservations/{id}`            |
| Evento Kafka `mesa-ya.tables.updated` | `tables.updated` + refresco list/detail | `internal/modules/realtime/application/handler` |
| Error de payload (`id` vacío)         | `{entity}.error` (`reason`)             | Validación en `executeDetailCommand`            |

## Checklist para nuevas entidades

1. **Dominios & comandos** → Define DTOs en `internal/modules/<entity>/domain`.
2. **Puertos REST** → Añade métodos a `port.SectionSnapshotFetcher` y su implementación HTTP.
3. **Use cases** → Implementa `List<Entity>`, `Get<Entity>` y registra `handleListCommand` / `handleDetailCommand`.
4. **Transport layer** → Añade fábrica en `entityHandlers` con alias de acción.
5. **Kafka** → Configura tópicos en `WS_ENTITY_TOPICS` y actualiza `docs/kafka-guide.md` si es necesario.
6. **Docs** → Actualiza esta guía y `docs/WEBSOCKET_ENDPOINTS.md` con rutas/comandos.
7. **Tests** → Cubrir normalización y flujos (ver `internal/modules/realtime/interface/http.handler_test.go` y `connect_section_test.go`).
8. **Swagger** → Asegura que los endpoints REST dependientes estén documentados para debugging.

## Recursos relacionados

- `docs/REALTIME_WEBSOCKET_GUIDE.md` – Arquitectura y extensión del módulo.
- `docs/WEBSOCKET_ENDPOINTS.md` – Rutas y ejemplos de conexión.
- `docs/kafka-guide.md` – Catálogo de eventos Kafka que alimentan al broadcaster.
- `docs/swagger/swagger.yml` – API completa para consultas REST.
- `internal/modules/realtime/application/usecase/connect_section.go` – Orquestación de comandos y caching.
- `internal/modules/realtime/interface/http.handler.go` – Adaptador Echo + Gorilla WebSocket.

Con esta guía cuentas con un panorama completo de los tópicos WebSocket en MesaYa, los comandos disponibles y su relación con la API REST y los eventos Kafka, permitiendo mantener integraciones consistentes y diagnósticos rápidos.
