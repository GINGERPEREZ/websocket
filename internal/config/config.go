package config

import (
	"os"
	"strings"
)

type Config struct {
	KafkaBrokers   []string
	ServerPort     string
	JWTSecret      string
	RestBaseURL    string
	EntityTopics   map[string]string
	AllowedActions []string
}

func Load() *Config {
	brokers := split(os.Getenv("KAFKA_BROKERS"))
	if len(brokers) == 0 {
		brokers = split(os.Getenv("KAFKA_BROKER"))
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	baseURL := strings.TrimSpace(os.Getenv("REST_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	topics := parseTopics(os.Getenv("WS_ENTITY_TOPICS"))
	if len(topics) == 0 {
		topics = map[string]string{
			"users":       "users.events",
			"restaurants": "restaurants.events",
			"orders":      "orders.events",
			"bookings":    "bookings.events",
			"sections":    "sections.events",
			"reviews":     "reviews.events",
		}
	}

	actions := split(os.Getenv("WS_ALLOWED_ACTIONS"))
	if len(actions) == 0 {
		actions = []string{"created", "updated", "deleted", "snapshot"}
	}

	return &Config{
		KafkaBrokers:   brokers,
		ServerPort:     port,
		JWTSecret:      secret,
		RestBaseURL:    baseURL,
		EntityTopics:   topics,
		AllowedActions: actions,
	}
}

func split(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func parseTopics(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	entries := strings.Split(raw, ",")
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
