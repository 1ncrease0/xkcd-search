package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel     string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	IndexTTL     time.Duration `yaml:"index_ttl" env:"INDEX_TTL" env-default:"24h"`
	Address      string        `yaml:"search_address" env:"SEARCH_ADDRESS" env-default:"localhost:8083"`
	DBAddress    string        `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`
	WordsAddress string        `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"localhost:8081"`
	Nats         Nats          `yaml:"nats"`
}

type Nats struct {
	Address     string `yaml:"address" env:"BROKER_ADDRESS" env-default:"nats://localhost:4222"`
	UpdateTopic string `yaml:"update_topic" env:"UPDATE_TOPIC" env-default:"xkcd.db.updated"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
