package port

import (
    "context"

    "mesaYaWs/internal/modules/realtime/domain"
)

// PubSubPort define el contrato para consumir eventos externos (Kafka).
type PubSubPort interface {
    Consume(ctx context.Context, topic string, handler func(*domain.Message) error) error
}

// Broadcaster define el contrato para enviar mensajes a los clientes WebSocket.
type Broadcaster interface {
    Broadcast(ctx context.Context, msg *domain.Message)
}

// TopicHandler define la interfaz que deben implementar los handlers registrados por tópico.
type TopicHandler interface {
    Topic() string
    Handle(ctx context.Context, msg *domain.Message) error
}
