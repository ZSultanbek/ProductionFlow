package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const pagesOrigin = "https://productionflow.pages.dev"

func corsRouter(cfg CORS) http.Handler {
	deps := testDeps(nil)
	deps.CORS = cfg
	return NewRouter(deps)
}

func request(h http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCORSAllowsAConfiguredOrigin(t *testing.T) {
	h := corsRouter(CORS{AllowedOrigins: []string{pagesOrigin}})

	rec := request(h, http.MethodGet, "/api/health", map[string]string{"Origin": pagesOrigin})

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != pagesOrigin {
		t.Errorf("Allow-Origin = %q, want %q", got, pagesOrigin)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin", got)
	}
	// The token is sent in a header, never a cookie.
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Allow-Credentials = %q, want it absent", got)
	}
}

func TestCORSIgnoresOtherOrigins(t *testing.T) {
	h := corsRouter(CORS{AllowedOrigins: []string{pagesOrigin}})

	for _, origin := range []string{
		"https://evil.example",
		"https://productionflow.pages.dev.evil.example",
		"http://productionflow.pages.dev", // wrong scheme
		"http://localhost:5000",           // localhost is not allowed unless asked for
	} {
		rec := request(h, http.MethodGet, "/api/health", map[string]string{"Origin": origin})
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("origin %q got Allow-Origin %q", origin, got)
		}
		// The request itself is still served; it is the browser that blocks
		// the page from reading it.
		if rec.Code != http.StatusOK {
			t.Errorf("origin %q: status %d, want 200", origin, rec.Code)
		}
	}
}

// The browser sends a preflight with no Authorization header. If it reached
// requireAuth it would get 401 and every logged-in call from the web app
// would fail before it was even sent.
func TestCORSPreflightIsAnsweredBeforeAuth(t *testing.T) {
	h := corsRouter(CORS{AllowedOrigins: []string{pagesOrigin}})

	rec := request(h, http.MethodOptions, "/api/me", map[string]string{
		"Origin":                         pagesOrigin,
		"Access-Control-Request-Method":  "GET",
		"Access-Control-Request-Headers": "authorization",
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	for header, want := range map[string]string{
		"Access-Control-Allow-Origin":  pagesOrigin,
		"Access-Control-Allow-Methods": "GET, POST, PUT, PATCH, DELETE",
		"Access-Control-Allow-Headers": "Authorization, Content-Type",
		"Access-Control-Max-Age":       "600",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestCORSPreflightFromUnknownOriginGetsNothing(t *testing.T) {
	h := corsRouter(CORS{AllowedOrigins: []string{pagesOrigin}})

	rec := request(h, http.MethodOptions, "/api/me", map[string]string{
		"Origin":                        "https://evil.example",
		"Access-Control-Request-Method": "GET",
	})
	if rec.Header().Get("Access-Control-Allow-Origin") != "" || rec.Header().Get("Access-Control-Allow-Methods") != "" {
		t.Errorf("unknown origin got CORS headers: %v", rec.Header())
	}
}

// An error response needs the header too. Without it the browser hides the
// body, and the web app sees "network error" instead of "log in again".
func TestCORSHeadersAreOnErrorResponsesToo(t *testing.T) {
	h := corsRouter(CORS{AllowedOrigins: []string{pagesOrigin}})

	rec := request(h, http.MethodGet, "/api/me", map[string]string{"Origin": pagesOrigin})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != pagesOrigin {
		t.Errorf("401 response has Allow-Origin %q, want %q", got, pagesOrigin)
	}
}

// `flutter run -d chrome` serves the app from a different random port each run.
func TestCORSAllowLocalhostAdmitsAnyLocalPort(t *testing.T) {
	h := corsRouter(CORS{AllowLocalhost: true})

	for origin, want := range map[string]bool{
		"http://localhost:53124":        true,
		"http://localhost":              true,
		"http://127.0.0.1:8081":         true,
		"https://localhost:53124":       false, // the dev server is plain http
		"http://localhost.evil.example": false,
		"http://evil.example/localhost": false,
		"http://192.168.1.20:53124":     false,
		"null":                          false,
	} {
		rec := request(h, http.MethodGet, "/api/health", map[string]string{"Origin": origin})
		if got := rec.Header().Get("Access-Control-Allow-Origin") != ""; got != want {
			t.Errorf("origin %q: allowed = %v, want %v", origin, got, want)
		}
	}
}

// Same-origin requests, curl and the Android app send no Origin header at all.
func TestCORSLeavesRequestsWithoutOriginAlone(t *testing.T) {
	h := corsRouter(CORS{AllowedOrigins: []string{pagesOrigin}, AllowLocalhost: true})

	rec := request(h, http.MethodGet, "/api/health", nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("status %d, Allow-Origin %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
