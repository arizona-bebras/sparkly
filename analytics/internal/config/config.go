package config

import (
	"fmt"
	"os"
	"strconv"
)

const DefaultRawPersistenceWorkers = 4

type Config struct {
	Port                  string
	RabbitMQURL           string
	RawPersistenceWorkers int
}

func Load() (Config, error) {
	cfg := Config{Port: os.Getenv("PORT"), RabbitMQURL: os.Getenv("RABBITMQ_URL"), RawPersistenceWorkers: DefaultRawPersistenceWorkers}
	if cfg.Port == "" {
		cfg.Port = "8081"
	}
	if value := os.Getenv("RAW_PERSISTENCE_WORKERS"); value != "" {
		count, err := strconv.Atoi(value)
		if err != nil || count < 1 {
			return Config{}, fmt.Errorf("RAW_PERSISTENCE_WORKERS must be a positive integer")
		}
		cfg.RawPersistenceWorkers = count
	}
	return cfg, nil
}
