package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultDatabaseURL matches the postgres service in docker-compose.yml.
const DefaultDatabaseURL = "postgres://productionflow:productionflow@localhost:5432/productionflow?sslmode=disable"

// devJWTSecret signs tokens when nothing else is configured, so `make dev` works
// out of the box. It is public — it is in this file — so Load refuses to use it
// outside dev.
const devJWTSecret = "dev-only-not-a-secret-dev-only-not-a-secret"

// minJWTSecretBytes: HS256 wants a key at least as long as its 256-bit output.
const minJWTSecretBytes = 32

type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// DatabaseURL is a libpq-style connection string.
	DatabaseURL string
	// ShutdownTimeout is how long in-flight requests get to finish on SIGTERM.
	ShutdownTimeout time.Duration
	// Env is "dev" or "prod"; dev switches the logger to text output.
	Env string
	// JWTSecret signs access tokens. Required outside dev.
	JWTSecret []byte
	// JWTTTL is how long an access token stays valid. No refresh flow in the
	// MVP, so this is also how long until the user has to log in again.
	JWTTTL time.Duration
	// PreviewTTL is how long a change preview can be confirmed for. Past it,
	// apply answers preview_expired and the producer looks again.
	PreviewTTL time.Duration
	// PublicWebURL is where the Flutter web app is served, without a trailing
	// slash. Invite links are built from it: {PublicWebURL}/join/{code}.
	PublicWebURL string
	// CORSAllowedOrigins are the browser origins allowed to call the API when
	// the web app is served from somewhere else (Cloudflare Pages in prod).
	CORSAllowedOrigins []string
	// CORSAllowLocalhost admits http://localhost on any port. True only in dev,
	// where `flutter run -d chrome` picks a new port every run.
	CORSAllowLocalhost bool
}

// Load reads the environment. A missing value takes its default; a value that
// is present but unparseable is an error. Outside dev, a missing JWT_SECRET is
// an error too — a production server signing tokens with a published key would
// let anyone log in as anyone.
func Load() (Config, error) {
	cfg := Config{
		Addr:            envString("API_ADDR", ":8080"),
		DatabaseURL:     envString("DATABASE_URL", DefaultDatabaseURL),
		ShutdownTimeout: 10 * time.Second,
		Env:             envString("APP_ENV", "dev"),
		JWTTTL:          7 * 24 * time.Hour,
		PreviewTTL:      10 * time.Minute,
		PublicWebURL:    strings.TrimRight(envString("PUBLIC_WEB_URL", "http://localhost:3000"), "/"),
	}

	cfg.CORSAllowLocalhost = cfg.Env == "dev"
	for _, origin := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		// Browsers send origins without a trailing slash; a configured one with
		// a slash would silently never match.
		if origin = strings.TrimRight(strings.TrimSpace(origin), "/"); origin != "" {
			cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, origin)
		}
	}

	if raw := os.Getenv("SHUTDOWN_TIMEOUT_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT_SECONDS: want a positive integer, got %q", raw)
		}
		cfg.ShutdownTimeout = time.Duration(seconds) * time.Second
	}

	if raw := os.Getenv("JWT_TTL_HOURS"); raw != "" {
		hours, err := strconv.Atoi(raw)
		if err != nil || hours <= 0 {
			return Config{}, fmt.Errorf("JWT_TTL_HOURS: want a positive integer, got %q", raw)
		}
		cfg.JWTTTL = time.Duration(hours) * time.Hour
	}

	if raw := os.Getenv("PREVIEW_TTL_MINUTES"); raw != "" {
		minutes, err := strconv.Atoi(raw)
		if err != nil || minutes <= 0 {
			return Config{}, fmt.Errorf("PREVIEW_TTL_MINUTES: want a positive integer, got %q", raw)
		}
		cfg.PreviewTTL = time.Duration(minutes) * time.Minute
	}

	switch secret := os.Getenv("JWT_SECRET"); {
	case secret != "" && len(secret) < minJWTSecretBytes:
		return Config{}, fmt.Errorf("JWT_SECRET: want at least %d bytes, got %d", minJWTSecretBytes, len(secret))
	case secret != "":
		cfg.JWTSecret = []byte(secret)
	case cfg.Env == "dev":
		cfg.JWTSecret = []byte(devJWTSecret)
	default:
		return Config{}, fmt.Errorf("JWT_SECRET is required when APP_ENV=%q (generate one with: openssl rand -base64 48)", cfg.Env)
	}

	return cfg, nil
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
