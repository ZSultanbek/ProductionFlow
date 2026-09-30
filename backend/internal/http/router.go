package http

import (
	"context"
	"log/slog"
	"net/http"
)

// Pinger is the slice of store.DB this package needs. Depending on the interface
// instead of the concrete type keeps the transport layer testable with a fake.
type Pinger interface {
	Ping(ctx context.Context) error
}

// BuildInfo is stamped in at link time by the Makefile and reported by /api/health.
type BuildInfo struct {
	Version string
	Commit  string
	BuiltAt string
}

// Deps is everything the handlers need, passed in from cmd/api.
type Deps struct {
	DB     Pinger
	Build  BuildInfo
	Logger *slog.Logger
}

// NewRouter registers the routes and wraps them in the middleware chain.
// Outermost first: a panic inside the logger's handler still gets recovered.
func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth(deps))

	// Contracted in docs/api.md, not implemented yet. Each of these returns 501.
	for _, pattern := range []string{
		"GET /api/projects",
		"POST /api/projects",
		"GET /api/days",
		"POST /api/days",
		"POST /api/changes/preview",
		"POST /api/changes/apply",
	} {
		mux.HandleFunc(pattern, notImplemented)
	}

	mux.HandleFunc("/", handleNotFound)

	return recoverPanic(deps.Logger)(requestLogger(deps.Logger)(mux))
}
