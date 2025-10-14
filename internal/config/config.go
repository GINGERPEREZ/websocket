package config

import "os"

type Config struct {
	KafkaBrokers []string
	ServerPort   string
}

func Load() *Config {
	return &Config{
		KafkaBrokers: []string{os.Getenv("KAFKA_BROKER")},
		ServerPort:   os.Getenv("PORT"),
	}
}
