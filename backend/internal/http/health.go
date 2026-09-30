package http

import (
	"context"
	"net/http"
	"runtime"
	"time"
)

// pingTimeout caps how long /api/health waits on Postgres. Health checks must
// answer quickly even when the database is wedged.
const pingTimeout = 2 * time.Second

type healthResponse struct {
	Status    string         `json:"status"`
	Version   string         `json:"version"`
	Commit    string         `json:"commit"`
	BuiltAt   string         `json:"built_at"`
	GoVersion string         `json:"go_version"`
	Database  databaseHealth `json:"database"`
}

type databaseHealth struct {
	Status    string  `json:"status"`
	LatencyMS float64 `json:"latency_ms"`
}

// handleHealth reports build info plus the result of a database ping. It returns
// 503 when the database is unreachable so a load balancer can act on it; the
// body shape is the same either way.
func handleHealth(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
		defer cancel()

		start := time.Now()
		pingErr := deps.DB.Ping(ctx)
		latency := time.Since(start)

		body := healthResponse{
			Status:    "ok",
			Version:   deps.Build.Version,
			Commit:    deps.Build.Commit,
			BuiltAt:   deps.Build.BuiltAt,
			GoVersion: runtime.Version(),
			Database: databaseHealth{
				Status:    "up",
				LatencyMS: float64(latency.Microseconds()) / 1000,
			},
		}

		status := http.StatusOK
		if pingErr != nil {
			// The reason stays in the log; the response must not leak driver errors.
			deps.Logger.Error("health: database ping failed", "error", pingErr)
			body.Status = "degraded"
			body.Database.Status = "down"
			status = http.StatusServiceUnavailable
		}

		writeJSON(w, status, body)
	}
}
