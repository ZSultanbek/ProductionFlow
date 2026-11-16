package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/auth"
)

// fakePinger stands in for store.DB — the reason NewRouter takes a Pinger.
type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func testDeps(pingErr error) Deps {
	projects := newFakeProjects()
	return Deps{
		DB:           fakePinger{err: pingErr},
		Users:        newFakeUsers(),
		Projects:     projects,
		Days:         newFakeDays(projects),
		Tokens:       auth.NewTokens([]byte("test-secret-test-secret-test-secret"), time.Hour),
		Build:        BuildInfo{Version: "test", Commit: "abc1234", BuiltAt: "2026-01-01T00:00:00Z"},
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		PublicWebURL: "https://web.test",
	}
}

func TestHealthDatabaseUp(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(testDeps(nil)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want %q", body.Status, "ok")
	}
	if body.Database.Status != "up" {
		t.Errorf("database.status = %q, want %q", body.Database.Status, "up")
	}
	if body.Version != "test" || body.Commit != "abc1234" {
		t.Errorf("build info not reported: %+v", body)
	}
	if body.GoVersion == "" {
		t.Error("go_version is empty")
	}
}

// A dead database must be visible as 503, not hidden behind a 200.
func TestHealthDatabaseDown(t *testing.T) {
	rec := httptest.NewRecorder()
	deps := testDeps(errors.New("dial tcp 127.0.0.1:5432: connect: connection refused"))
	NewRouter(deps).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
	}
	if body.Status != "degraded" {
		t.Errorf("status = %q, want %q", body.Status, "degraded")
	}
	if body.Database.Status != "down" {
		t.Errorf("database.status = %q, want %q", body.Database.Status, "down")
	}
	// The driver error stays in the log: it must not reach the client.
	if raw := rec.Body.String(); strings.Contains(raw, "connection refused") {
		t.Errorf("response leaked the driver error: %s", raw)
	}
}
