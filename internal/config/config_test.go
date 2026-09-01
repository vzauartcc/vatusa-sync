package config

import (
	"testing"
)

func TestLoadParsesEnvironment(t *testing.T) {
	t.Setenv("ZAU_API_KEY", "zau-key")
	t.Setenv("ZAU_API_URL", "https://zau.example.com")
	t.Setenv("VATUSA_API_KEY", "vat-key")
	t.Setenv("VATUSA_API_URL", "https://vatusa.example.com")
	t.Setenv("LOCAL_DEV_ENVIRONMENT", "true")

	cfg := Load()

	if cfg.ZauAPIKey != "zau-key" || cfg.ZauAPIURL != "https://zau.example.com" {
		t.Errorf("unexpected ZAU config: %+v", cfg)
	}

	if cfg.VatusaAPIKey != "vat-key" || cfg.VatusaAPIURL != "https://vatusa.example.com" {
		t.Errorf("unexpected VATUSA config: %+v", cfg)
	}

	if cfg.LocalDevEnv != "true" {
		t.Errorf("unexpected local dev env: %q", cfg.LocalDevEnv)
	}
}
