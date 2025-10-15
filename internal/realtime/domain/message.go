package domain

import "time"

// Message representa el mensaje de dominio que se transmite entre Kafka y WebSocket.
// Topic corresponde al canal final de WebSocket (entity.action) mientras que Entity y Action
// describen el evento del dominio. Metadata permite incluir información adicional (ej. userId destino).
type Message struct {
	Topic      string            `json:"topic"`
	Entity     string            `json:"entity"`
	Action     string            `json:"action"`
	ResourceID string            `json:"resourceId,omitempty"`
	Data       interface{}       `json:"data,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}
