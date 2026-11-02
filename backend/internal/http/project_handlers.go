package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// ProjectStore is the slice of store.DB the project and invite handlers need.
type ProjectStore interface {
	CreateProject(ctx context.Context, owner domain.User, name string) (domain.Membership, error)
	ProjectsForUser(ctx context.Context, userID string, after *store.Keyset, limit int) ([]domain.Membership, error)
	ProjectForUser(ctx context.Context, userID, projectID string) (domain.Membership, error)
	ProjectByInviteCode(ctx context.Context, code string) (domain.Project, error)
	JoinProject(ctx context.Context, user domain.User, projectID string, role domain.Role) (domain.Participant, bool, error)
}

// --- wire shapes (docs/api.md, "Shared shapes") ---

type projectJSON struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	MyRole          string  `json:"my_role"`
	MyParticipantID string  `json:"my_participant_id"`
	InviteCode      *string `json:"invite_code"`
	InviteURL       *string `json:"invite_url"`
	CreatedAt       string  `json:"created_at"`
}

// toProjectJSON shows the invite code to producers only. Anyone holding the
// code can join, so handing it to every crew member would make "who can invite"
// mean "everyone".
func toProjectJSON(m domain.Membership, publicWebURL string) projectJSON {
	out := projectJSON{
		ID:              m.Project.ID,
		Name:            m.Project.Name,
		MyRole:          string(m.Role),
		MyParticipantID: m.ParticipantID,
		CreatedAt:       formatInstant(m.Project.CreatedAt),
	}
	if m.Role == domain.RoleProducer {
		code := m.Project.InviteCode
		url := publicWebURL + "/join/" + code
		out.InviteCode, out.InviteURL = &code, &url
	}
	return out
}

func membershipKey(m domain.Membership) store.Keyset {
	return store.Keyset{CreatedAt: m.Project.CreatedAt, ID: m.Project.ID}
}

// --- handlers ---

// handleListProjects: GET /api/projects — the caller's projects, newest first.
func handleListProjects(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		after, limit, fieldErrs := parsePage(r)
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		list, err := deps.Projects.ProjectsForUser(r.Context(), userIDFrom(r.Context()), after, limit+1)
		if err != nil {
			writeInternalError(w, deps.Logger, "list projects", err)
			return
		}

		list, next := page(list, limit, membershipKey)
		data := make([]projectJSON, 0, len(list))
		for _, m := range list {
			data = append(data, toProjectJSON(m, deps.PublicWebURL))
		}
		writeJSON(w, http.StatusOK, listResponse[projectJSON]{Data: data, NextCursor: next})
	}
}

// handleCreateProject: POST /api/projects — the caller becomes its producer.
func handleCreateProject(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		name, fieldErrs := domain.ValidateProjectName(req.Name)
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		user, ok := currentUser(w, r, deps)
		if !ok {
			return
		}

		m, err := deps.Projects.CreateProject(r.Context(), user, name)
		if err != nil {
			writeInternalError(w, deps.Logger, "create project", err)
			return
		}
		writeJSON(w, http.StatusCreated, toProjectJSON(m, deps.PublicWebURL))
	}
}

// handleGetProject: GET /api/projects/{project_id}.
func handleGetProject(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := deps.Projects.ProjectForUser(r.Context(), userIDFrom(r.Context()), r.PathValue("project_id"))
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "project_not_found", "no such project")
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "get project", err)
			return
		}
		writeJSON(w, http.StatusOK, toProjectJSON(m, deps.PublicWebURL))
	}
}

// currentUser loads the caller's account. A valid token whose account has been
// deleted gets the same 401 as a bad token. On false a response has been written.
func currentUser(w http.ResponseWriter, r *http.Request, deps Deps) (domain.User, bool) {
	user, err := deps.Users.UserByID(r.Context(), userIDFrom(r.Context()))
	if errors.Is(err, store.ErrNotFound) {
		writeUnauthorized(w)
		return domain.User{}, false
	}
	if err != nil {
		writeInternalError(w, deps.Logger, "load current user", err)
		return domain.User{}, false
	}
	return user, true
}
