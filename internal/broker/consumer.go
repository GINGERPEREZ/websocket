package broker

import (
	"context"
	"encoding/json"
	"log"
	"mesaYaWs/internal/realtime/domain"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(brokers []string, groupID string, topic string) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			GroupID: groupID,
			Topic:   topic,
		}),
	}
}

func (c *KafkaConsumer) Consume(ctx context.Context, handler func(*domain.Message) error) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("kafka read error: %v", err)
			continue
		}
		msg := decodeMessage(m)
		if err := handler(msg); err != nil {
			log.Printf("handler error: %v", err)
		}
	}
}

type rawEvent struct {
	Entity     string            `json:"entity"`
	Action     string            `json:"action"`
	ResourceID string            `json:"resourceId"`
	Topic      string            `json:"topic"`
	Metadata   map[string]string `json:"metadata"`
	Data       interface{}       `json:"data"`
}

func decodeMessage(m kafka.Message) *domain.Message {
	msg := &domain.Message{Timestamp: time.Now().UTC()}

	var event rawEvent
	if err := json.Unmarshal(m.Value, &event); err != nil {
		msg.Topic = m.Topic
		msg.Entity = normalizeTopic(m.Topic)
		msg.Action = "raw"
		msg.Data = string(m.Value)
		return msg
	}

	msg.Entity = firstNonEmpty(event.Entity, normalizeTopic(m.Topic))
	msg.Action = firstNonEmpty(event.Action, "unknown")
	msg.ResourceID = event.ResourceID
	msg.Metadata = event.Metadata
	msg.Data = event.Data

	if event.Topic != "" {
		msg.Topic = event.Topic
	} else {
		msg.Topic = msg.Entity + "." + msg.Action
	}

	return msg
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func normalizeTopic(topic string) string {
	if idx := strings.LastIndex(topic, "."); idx >= 0 {
		topic = topic[idx+1:]
	}
	return strings.TrimSpace(topic)
}
