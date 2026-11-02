package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// The error envelope is a contract with the Flutter client: any failure is
// {"error":{"code":..., "message":...}} and nothing else. This test is what
// stops that shape drifting.
func TestWriteErrorEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, http.StatusNotFound, "day_not_found", "no shooting day with that id")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if got, want := rec.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	// Decoded into a map rather than errorResponse, so a renamed JSON tag fails
	// the test instead of silently round-tripping.
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
	}

	wrapped, ok := body["error"]
	if !ok {
		t.Fatalf("body has no \"error\" key: %v", body)
	}
	if got, want := wrapped["code"], "day_not_found"; got != want {
		t.Errorf("error.code = %q, want %q", got, want)
	}
	if got, want := wrapped["message"], "no shooting day with that id"; got != want {
		t.Errorf("error.message = %q, want %q", got, want)
	}
	if len(body) != 1 || len(wrapped) != 2 {
		t.Errorf("envelope has extra fields: %v", body)
	}
}

func TestStubsReturnNotImplemented(t *testing.T) {
	router := NewRouter(testDeps(nil))

	// Fill every {wildcard} with a UUID-shaped value to get a concrete path.
	wildcard := regexp.MustCompile(`\{[a-z_]+\}`)

	for _, pattern := range notYetImplemented {
		method, path, _ := strings.Cut(pattern, " ")
		path = wildcard.ReplaceAllString(path, "6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f")

		t.Run(pattern, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))

			if rec.Code != http.StatusNotImplemented {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
			}
			var body errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
			}
			if body.Error.Code != "not_implemented" {
				t.Errorf("error.code = %q, want %q", body.Error.Code, "not_implemented")
			}
		})
	}
}

func TestUnknownPathReturnsEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(testDeps(nil)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "not_found")
	}
}
