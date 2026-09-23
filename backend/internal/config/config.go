package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds every knob the service reads at startup. Everything comes from
// the environment so the binary is identical in dev, CI and prod.
type Config struct {
	Port            string
	DatabaseURL     string
	AllowedOrigins  []string
	ShutdownTimeout time.Duration
}

// Load reads configuration from the environment, applying dev-friendly
// defaults. It returns an error rather than panicking so main() decides how
// to fail.
func Load() (Config, error) {
	cfg := Config{
		Port:            env("PORT", "8080"),
		DatabaseURL:     env("DATABASE_URL", ""),
		ShutdownTimeout: 10 * time.Second,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required (copy .env.example to .env)")
	}

	// CORS_ALLOWED_ORIGINS is a comma-separated list. The Vite dev server runs
	// on 5173 by default, so that is the out-of-the-box value.
	raw := env("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")
	for _, origin := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, trimmed)
		}
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
