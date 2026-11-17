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
		"JWT_SECRET", "JWT_TTL_HOURS", "PREVIEW_TTL_MINUTES", "PUBLIC_WEB_URL", "CORS_ALLOWED_ORIGINS",
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
	if cfg.PreviewTTL != 10*time.Minute {
		t.Errorf("PreviewTTL = %v, want 10 minutes (docs/api.md promises it)", cfg.PreviewTTL)
	}
}

func TestLoadPreviewTTL(t *testing.T) {
	clearEnv(t)
	t.Setenv("PREVIEW_TTL_MINUTES", "3")
	cfg, err := Load()
	if err != nil || cfg.PreviewTTL != 3*time.Minute {
		t.Errorf("PreviewTTL = %v, err = %v; want 3m", cfg.PreviewTTL, err)
	}

	t.Setenv("PREVIEW_TTL_MINUTES", "0")
	if _, err := Load(); err == nil {
		t.Error("accepted PREVIEW_TTL_MINUTES=0")
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

func TestLoadPublicWebURL(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicWebURL != "http://localhost:3000" {
		t.Errorf("default = %q", cfg.PublicWebURL)
	}

	// A trailing slash would produce invite links with "//join".
	t.Setenv("PUBLIC_WEB_URL", "https://productionflow.pages.dev/")
	if cfg, _ = Load(); cfg.PublicWebURL != "https://productionflow.pages.dev" {
		t.Errorf("trailing slash not trimmed: %q", cfg.PublicWebURL)
	}
}

func TestLoadCORS(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.CORSAllowLocalhost || len(cfg.CORSAllowedOrigins) != 0 {
		t.Errorf("dev defaults: localhost=%v origins=%v; want localhost allowed, no origins",
			cfg.CORSAllowLocalhost, cfg.CORSAllowedOrigins)
	}

	t.Setenv("APP_ENV", "prod")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 48))
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://productionflow.pages.dev/ , https://example.kz,, ")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	// Localhost must never be open in production.
	if cfg.CORSAllowLocalhost {
		t.Error("prod allows localhost origins")
	}
	want := []string{"https://productionflow.pages.dev", "https://example.kz"}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != want[0] || cfg.CORSAllowedOrigins[1] != want[1] {
		t.Errorf("origins = %q, want %q", cfg.CORSAllowedOrigins, want)
	}
}
