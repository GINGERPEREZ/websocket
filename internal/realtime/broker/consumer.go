package broker

import (
	"context"
	"log"
	"mesaYaWs/internal/realtime/domain"
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
		msg := &domain.Message{
			Topic:     m.Topic,
			Payload:   string(m.Value),
			Timestamp: time.Now(),
		}
		if err := handler(msg); err != nil {
			log.Printf("handler error: %v", err)
		}
	}
}
