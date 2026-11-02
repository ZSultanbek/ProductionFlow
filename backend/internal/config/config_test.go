package config

import (
	"strings"
	"testing"
	"time"
)

// clearEnv blanks every variable Load reads, so the developer's own shell
// cannot change the outcome of a test.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"API_ADDR", "DATABASE_URL", "APP_ENV", "SHUTDOWN_TIMEOUT_SECONDS",
		"JWT_SECRET", "JWT_TTL_HOURS",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadDevDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if string(cfg.JWTSecret) != devJWTSecret {
		t.Error("dev should fall back to the dev secret")
	}
	if cfg.JWTTTL != 7*24*time.Hour {
		t.Errorf("JWTTTL = %v, want 7 days", cfg.JWTTTL)
	}
}

// The one check in this package that protects production.
func TestLoadProdRequiresSecret(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "prod")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("prod started without JWT_SECRET (err = %v)", err)
	}
}

func TestLoadRejectsShortSecret(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "too-short")

	if _, err := Load(); err == nil {
		t.Fatal("accepted a 9-byte JWT_SECRET")
	}
}

func TestLoadReadsSecretAndTTL(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "prod")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 48))
	t.Setenv("JWT_TTL_HOURS", "24")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if string(cfg.JWTSecret) != strings.Repeat("s", 48) {
		t.Error("JWT_SECRET not used")
	}
	if cfg.JWTTTL != 24*time.Hour {
		t.Errorf("JWTTTL = %v, want 24h", cfg.JWTTTL)
	}
}

func TestLoadRejectsBadTTL(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_TTL_HOURS", "a week")

	if _, err := Load(); err == nil {
		t.Fatal("accepted a non-numeric JWT_TTL_HOURS")
	}
}
