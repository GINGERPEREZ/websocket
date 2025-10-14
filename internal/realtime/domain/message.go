package domain

import "time"

// Message representa el mensaje de dominio que se transmite entre Kafka y WebSocket.
type Message struct {
	Topic     string      `json:"topic"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}
