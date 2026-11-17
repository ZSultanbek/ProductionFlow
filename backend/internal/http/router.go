package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/auth"
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
	DB       Pinger
	Users    UserStore
	Projects ProjectStore
	Days     DayStore
	Scenes   SceneStore
	Tokens   *auth.Tokens
	// Previews signs the token a change preview hands out, which apply checks.
	Previews *auth.Previews
	Build    BuildInfo
	Logger   *slog.Logger
	// PublicWebURL is the web app's base URL, used to build invite links.
	PublicWebURL string
	// CORS lists the browser origins allowed to call the API. The zero value
	// allows none, which is right whenever web and API share an origin.
	CORS CORS
}

// notYetImplemented is every route in docs/api.md that has no handler yet. Each
// returns 501 so the client can point at a real server and get a parseable
// envelope. Move a line out of here when its handler lands, and update the
// Status table in docs/api.md in the same PR.
var notYetImplemented = []string{
	"POST /api/changes/apply",
	"GET /api/projects/{project_id}/changes",
	"GET /api/me/schedule",
	"GET /api/me/notifications",
	"POST /api/me/notifications/{notification_id}/read",
}

// NewRouter registers the routes and wraps them in the middleware chain.
// Outermost first: a panic inside the logger's handler still gets recovered.
func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	// Public.
	mux.HandleFunc("GET /api/health", handleHealth(deps))
	mux.HandleFunc("POST /api/auth/register", handleRegister(deps))
	mux.HandleFunc("POST /api/auth/login", handleLogin(deps))
	mux.HandleFunc("GET /api/invites/{code}", handleGetInvite(deps))

	// Logged in. Every new route goes below this line, wrapped in requireAuth,
	// unless docs/api.md marks it *public*.
	mux.HandleFunc("GET /api/me", requireAuth(deps, handleMe(deps)))

	mux.HandleFunc("GET /api/projects", requireAuth(deps, handleListProjects(deps)))
	mux.HandleFunc("POST /api/projects", requireAuth(deps, handleCreateProject(deps)))
	mux.HandleFunc("GET /api/projects/{project_id}", requireAuth(deps, handleGetProject(deps)))
	mux.HandleFunc("POST /api/invites/{code}/join", requireAuth(deps, handleJoinInvite(deps)))

	mux.HandleFunc("GET /api/projects/{project_id}/days", requireAuth(deps, handleListDays(deps)))
	mux.HandleFunc("POST /api/projects/{project_id}/days", requireAuth(deps, handleCreateDay(deps)))
	mux.HandleFunc("GET /api/days/{day_id}", requireAuth(deps, handleGetDay(deps)))
	mux.HandleFunc("GET /api/me/today", requireAuth(deps, handleToday(deps)))

	mux.HandleFunc("GET /api/projects/{project_id}/participants", requireAuth(deps, handleListParticipants(deps)))
	mux.HandleFunc("POST /api/projects/{project_id}/participants", requireAuth(deps, handleCreateParticipant(deps)))
	mux.HandleFunc("PUT /api/days/{day_id}/participants", requireAuth(deps, handleSetDayParticipants(deps)))

	mux.HandleFunc("GET /api/projects/{project_id}/scenes", requireAuth(deps, handleListScenes(deps)))
	mux.HandleFunc("POST /api/projects/{project_id}/scenes", requireAuth(deps, handleCreateScene(deps)))
	mux.HandleFunc("PATCH /api/scenes/{scene_id}", requireAuth(deps, handleUpdateScene(deps)))
	mux.HandleFunc("PUT /api/scenes/{scene_id}/participants", requireAuth(deps, handleSetSceneParticipants(deps)))

	mux.HandleFunc("POST /api/changes/preview", requireAuth(deps, handlePreviewChange(deps)))

	for _, pattern := range notYetImplemented {
		mux.HandleFunc(pattern, notImplemented)
	}

	mux.HandleFunc("/", handleNotFound)

	// cors sits outside the mux so a preflight is answered before any route or
	// requireAuth sees it, and inside the logger so preflights are logged too.
	return recoverPanic(deps.Logger)(requestLogger(deps.Logger)(cors(deps.CORS)(mux)))
}
