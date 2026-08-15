package config

import (
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ZauAPIKey string `env:"ZAU_API_KEY,notEmpty"`
	ZauAPIURL string `env:"ZAU_API_URL,notEmpty"`

	VatusaAPIKey string `env:"VATUSA_API_KEY,notEmpty"`
	VatusaAPIURL string `env:"VATUSA_API_URL,notEmpty"`

	LocalDevEnv string `env:"LOCAL_DEV_ENVIRONMENT"`
}

func Load() *Config {
	cfg := &Config{}

	err := env.Parse(cfg)
	if err != nil {
		slog.Error("failed to parse environment variables", "error", err)
		os.Exit(1)
	}

	return cfg
}
