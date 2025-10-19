package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config groups the runtime configuration of the realtime service following
// Clean Architecture boundaries: infrastructure (Kafka/REST), application
// concerns (websocket defaults), and cross-cutting concerns (logging/security).
type Config struct {
	Server    ServerConfig
	Kafka     KafkaConfig
	Security  SecurityConfig
	REST      RESTConfig
	Logging   LoggingConfig
	Websocket WebsocketConfig
}

type ServerConfig struct {
	Port string
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
	Topics  map[string]string
}

type SecurityConfig struct {
	JWTSecret string
}

type RESTConfig struct {
	BaseURL string
	Timeout time.Duration
}

type LoggingConfig struct {
	Directory string
	Level     string
	Format    string
}

type WebsocketConfig struct {
	AllowedActions []string
	DefaultEntity  string
}

// Load builds the Config from environment variables applying sensible defaults.
func Load() (Config, error) {
	cfg := Config{
		Server: ServerConfig{Port: stringOrDefault(os.Getenv("PORT"), "8080")},
		Kafka: KafkaConfig{
			Brokers: firstNonEmptySlice(splitEnv(os.Getenv("KAFKA_BROKERS")), splitEnv(os.Getenv("KAFKA_BROKER"))),
			GroupID: stringOrDefault(os.Getenv("KAFKA_GROUP_ID"), "realtime-group"),
			Topics:  parseTopics(os.Getenv("WS_ENTITY_TOPICS")),
		},
		Security: SecurityConfig{JWTSecret: trimQuotes(os.Getenv("JWT_SECRET"))},
		REST: RESTConfig{
			BaseURL: stringOrDefault(trimQuotes(os.Getenv("REST_BASE_URL")), "http://localhost:3000"),
			Timeout: durationOrDefault(os.Getenv("REST_TIMEOUT"), 10*time.Second),
		},
		Logging: LoggingConfig{
			Directory: stringOrDefault(trimQuotes(os.Getenv("LOG_DIR")), "./logs"),
			Level:     strings.TrimSpace(os.Getenv("LOG_LEVEL")),
			Format:    stringOrDefault(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "json"),
		},
		Websocket: WebsocketConfig{
			AllowedActions: firstNonEmptySlice(splitEnv(os.Getenv("WS_ALLOWED_ACTIONS")), []string{"created", "updated", "deleted", "snapshot"}),
			DefaultEntity:  stringOrDefault(strings.TrimSpace(os.Getenv("WS_DEFAULT_ENTITY")), "restaurants"),
		},
	}

	if len(cfg.Kafka.Topics) == 0 {
		cfg.Kafka.Topics = map[string]string{
			"users":       "users.events",
			"restaurants": "restaurants.events",
			"orders":      "orders.events",
			"bookings":    "bookings.events",
			"sections":    "sections.events",
			"reviews":     "reviews.events",
		}
	}

	if len(cfg.Kafka.Brokers) == 0 {
		cfg.Kafka.Brokers = []string{"localhost:9092"}
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.Security.JWTSecret == "" {
		return errors.New("jwt secret is required")
	}
	if _, err := urlFromString(c.REST.BaseURL); err != nil {
		return fmt.Errorf("invalid REST_BASE_URL: %w", err)
	}
	return nil
}

func splitEnv(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func firstNonEmptySlice(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func stringOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func trimQuotes(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) >= 2 {
		head := trimmed[0]
		tail := trimmed[len(trimmed)-1]
		if (head == '"' && tail == '"') || (head == '\'' && tail == '\'') {
			return strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		}
	}
	return trimmed
}

func parseTopics(raw string) map[string]string {
	entries := splitEnv(raw)
	if len(entries) == 0 {
		return nil
	}
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}
		entity := strings.TrimSpace(parts[0])
		topic := strings.TrimSpace(parts[1])
		if entity == "" || topic == "" {
			continue
		}
		result[entity] = topic
	}
	return result
}

func durationOrDefault(raw string, fallback time.Duration) time.Duration {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	if d, err := time.ParseDuration(trimmed); err == nil {
		return d
	}
	if seconds, err := strconv.Atoi(trimmed); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func urlFromString(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty url")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("missing scheme or host")
	}
	return parsed, nil
}
