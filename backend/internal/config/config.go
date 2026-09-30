package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DefaultDatabaseURL matches the postgres service in docker-compose.yml.
const DefaultDatabaseURL = "postgres://productionflow:productionflow@localhost:5432/productionflow?sslmode=disable"

type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// DatabaseURL is a libpq-style connection string.
	DatabaseURL string
	// ShutdownTimeout is how long in-flight requests get to finish on SIGTERM.
	ShutdownTimeout time.Duration
	// Env is "dev" or "prod"; dev switches the logger to text output.
	Env string
}

// Load reads the environment. It only fails on a value that is present but
// unparseable — a missing value is not an error, it takes the default.
func Load() (Config, error) {
	cfg := Config{
		Addr:            envString("API_ADDR", ":8080"),
		DatabaseURL:     envString("DATABASE_URL", DefaultDatabaseURL),
		ShutdownTimeout: 10 * time.Second,
		Env:             envString("APP_ENV", "dev"),
	}

	if raw := os.Getenv("SHUTDOWN_TIMEOUT_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT_SECONDS: want a positive integer, got %q", raw)
		}
		cfg.ShutdownTimeout = time.Duration(seconds) * time.Second
	}

	return cfg, nil
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
