# 🔌 MesaYA - WebSocket Service

Servicio de comunicación en tiempo real para la plataforma MesaYA, construido con Go.

## 📋 Descripción

Este microservicio proporciona comunicación bidireccional en tiempo real usando WebSockets para:

- **Notificaciones en tiempo real**: Alertas instantáneas sobre reservas, cancelaciones, etc.
- **Actualizaciones de disponibilidad**: Cambios en el estado de mesas en vivo
- **Chat en vivo**: Comunicación entre clientes y restaurantes
- **Sincronización de datos**: Actualizaciones automáticas en todas las sesiones activas
- **Integración con Kafka**: Consume eventos del sistema para notificar a clientes conectados

## 🏗️ Arquitectura

```
cmd/
└── server/
    └── main.go          # Punto de entrada de la aplicación

internal/
├── config/              # Configuración de la aplicación
├── handlers/            # Manejadores de WebSocket
├── kafka/               # Cliente de Kafka
├── models/              # Estructuras de datos
└── websocket/           # Lógica de WebSocket
```

## 🚀 Instalación y Ejecución

### Prerrequisitos

- Go 1.21+
- Kafka (debe estar corriendo)

### Instalación

```bash
# Clonar el proyecto (si no lo tienes)
cd mesaYA_ws

# Descargar dependencias
go mod download
```

### Variables de Entorno

Crear un archivo `.env` con las siguientes variables:

```env
# Server
PORT=8080
HOST=0.0.0.0

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=mesaya-ws-group

# CORS
ALLOWED_ORIGINS=http://localhost:4200,http://localhost:3000
```

### Ejecutar

```bash
# Modo desarrollo
go run ./cmd/server/main.go

# Compilar
go build -o server ./cmd/server/main.go

# Ejecutar compilado
./server

# Con Docker
docker compose up -d
```

## 📡 Uso del WebSocket

### Conexión desde el cliente

```javascript
// Conectar al WebSocket
const ws = new WebSocket('ws://localhost:8080/ws');

// Escuchar mensajes
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Mensaje recibido:', data);
};

// Enviar mensajes
ws.send(JSON.stringify({
  type: 'subscribe',
  channel: 'restaurant:123:reservations'
}));

// Manejar errores
ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

// Reconectar al cerrar
ws.onclose = () => {
  console.log('Conexión cerrada, reconectando...');
  setTimeout(() => {
    // Lógica de reconexión
  }, 3000);
};
```

## 📬 Tipos de Mensajes

### Cliente → Servidor

```json
{
  "type": "subscribe",
  "channel": "restaurant:123:reservations"
}
```

```json
{
  "type": "unsubscribe",
  "channel": "restaurant:123:reservations"
}
```

### Servidor → Cliente

**Nueva Reserva:**

```json
{
  "type": "reservation.created",
  "data": {
    "reservationId": "abc123",
    "restaurantId": "123",
    "tableId": "456",
    "clientName": "Juan Pérez",
    "date": "2026-01-20T19:00:00Z"
  }
}
```

**Cambio de Estado:**

```json
{
  "type": "reservation.status_changed",
  "data": {
    "reservationId": "abc123",
    "newStatus": "confirmed",
    "previousStatus": "pending"
  }
}
```

**Actualización de Mesa:**

```json
{
  "type": "table.updated",
  "data": {
    "tableId": "456",
    "status": "available",
    "capacity": 4
  }
}
```

## 🔔 Canales de Suscripción

Los clientes pueden suscribirse a diferentes canales:

- `restaurant:{id}:reservations` - Todas las reservas de un restaurante
- `restaurant:{id}:tables` - Estado de mesas de un restaurante
- `user:{id}:notifications` - Notificaciones de un usuario específico
- `global:announcements` - Anuncios globales del sistema

## 🧪 Testing

```bash
# Ejecutar tests
go test ./...

# Con cobertura
go test -cover ./...

# Test específico
go test ./internal/websocket

# Con verbose
go test -v ./...
```

## 🛠️ Tecnologías

- **Go (Golang)** - Lenguaje de programación
- **Gorilla WebSocket** - Implementación de WebSocket para Go
- **Sarama** - Cliente de Kafka para Go
- **Godotenv** - Gestión de variables de entorno
- **CORS** - Manejo de políticas CORS

## 📊 Características Técnicas

- **Alta concurrencia**: Goroutines para manejar múltiples conexiones simultáneas
- **Baja latencia**: Comunicación directa sin polling
- **Escalable**: Diseñado para manejar miles de conexiones
- **Resiliente**: Reconexión automática y manejo de errores
- **Event-driven**: Integrado con Kafka para recibir eventos del sistema

## 🔍 Monitoreo

El servicio expone endpoints para monitoreo:

- `GET /health` - Health check
- `GET /metrics` - Métricas del servicio (conexiones activas, mensajes enviados, etc.)

## 📚 Más Información

Para más detalles sobre la arquitectura y funcionamiento del sistema completo, consulta la [documentación principal](../docs/).

## 📄 Licencia

Este proyecto es parte de MesaYA y está desarrollado por estudiantes de ULEAM.
