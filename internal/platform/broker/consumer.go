package broker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"mesaYaWs/internal/modules/realtime/domain"
)

// Circuit breaker constants
const (
	maxConsecutiveErrors = 3
	initialBackoff       = 5 * time.Second
	maxBackoff           = 60 * time.Second
	logThrottleInterval  = 30 * time.Second
)

// Global circuit breaker shared across all consumers
var globalCircuit = &circuitBreaker{
	currentBackoff: initialBackoff,
}

type circuitBreaker struct {
	mu               sync.Mutex
	consecutiveErrs  int
	currentBackoff   time.Duration
	lastErrorLogTime time.Time
	circuitOpen      bool
}

func (cb *circuitBreaker) isOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.circuitOpen
}

func (cb *circuitBreaker) recordError(err error, topic string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveErrs++

	// Only log periodically to avoid spam
	now := time.Now()
	if now.Sub(cb.lastErrorLogTime) >= logThrottleInterval {
		slog.Warn("kafka connection error",
			slog.Any("error", err),
			slog.String("topic", topic),
			slog.Int("consecutive_errors", cb.consecutiveErrs),
			slog.Duration("backoff", cb.currentBackoff),
		)
		cb.lastErrorLogTime = now
	}

	// Open circuit after max consecutive errors
	if cb.consecutiveErrs >= maxConsecutiveErrors && !cb.circuitOpen {
		cb.circuitOpen = true
		slog.Info("kafka circuit breaker OPEN - will retry in",
			slog.Duration("backoff", cb.currentBackoff),
		)
	}
}

func (cb *circuitBreaker) waitBackoff(ctx context.Context) {
	cb.mu.Lock()
	backoff := cb.currentBackoff
	cb.mu.Unlock()

	select {
	case <-ctx.Done():
		return
	case <-time.After(backoff):
		cb.mu.Lock()
		cb.circuitOpen = false
		// Exponential backoff for next failure
		cb.currentBackoff *= 2
		if cb.currentBackoff > maxBackoff {
			cb.currentBackoff = maxBackoff
		}
		cb.mu.Unlock()
		slog.Info("kafka circuit breaker CLOSED - retrying connection")
	}
}

func (cb *circuitBreaker) reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.consecutiveErrs > 0 {
		slog.Info("kafka connection restored")
	}
	cb.consecutiveErrs = 0
	cb.currentBackoff = initialBackoff
	cb.circuitOpen = false
}

type KafkaConsumer struct {
	reader *kafka.Reader
	topic  string
}

func NewKafkaConsumer(brokers []string, groupID string, topic string) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			GroupID: groupID,
			Topic:   topic,
		}),
		topic: topic,
	}
}

func (c *KafkaConsumer) Consume(ctx context.Context, handler func(*domain.Message) error) error {
	for {
		// Check if global circuit is open
		if globalCircuit.isOpen() {
			globalCircuit.waitBackoff(ctx)
			continue
		}

		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			globalCircuit.recordError(err, c.topic)
			continue
		}

		// Reset global circuit on success
		globalCircuit.reset()
		msg := decodeMessage(m)
		slog.Info("kafka message consumed",
			slog.String("topic", m.Topic),
			slog.Int("partition", m.Partition),
			slog.Int64("offset", m.Offset),
			slog.String("entity", msg.Entity),
			slog.String("action", msg.Action),
			slog.String("resourceId", msg.ResourceID),
			slog.Any("metadata", msg.Metadata),
		)
		if err := handler(msg); err != nil {
			slog.Warn("kafka handler error", slog.Any("error", err))
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
		entity, action := inferEntityActionFromTopic(m.Topic)
		msg.Entity = entity
		msg.Action = action
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

func inferEntityActionFromTopic(topic string) (string, string) {
	parts := strings.Split(topic, ".")
	if len(parts) >= 2 {
		entity := strings.TrimSpace(parts[len(parts)-2])
		action := strings.TrimSpace(parts[len(parts)-1])
		if entity != "" && action != "" {
			return entity, action
		}
	}
	if entity := normalizeTopic(topic); entity != "" {
		return entity, "unknown"
	}
	return "", "unknown"
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
