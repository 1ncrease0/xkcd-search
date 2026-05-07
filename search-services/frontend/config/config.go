package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPConfig struct {
	Address string        `yaml:"address" env:"FRONTEND_ADDRESS" env-default:":8089"`
	Timeout time.Duration `yaml:"timeout" env:"FRONTEND_TIMEOUT" env-default:"5s"`
}

type Config struct {
	LogLevel      string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	HTTPConfig    HTTPConfig    `yaml:"frontend_server"`
	APIBaseURL    string        `yaml:"api_url" env:"API_URL" env-default:"http://localhost:8080"`
	ClientTimeout time.Duration `yaml:"client_timeout" env:"CLIENT_TIMEOUT" env-default:"120s"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
