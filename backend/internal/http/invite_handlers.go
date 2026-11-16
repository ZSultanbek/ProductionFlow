package http

import (
	"errors"
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

type participantJSON struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	DisplayName string  `json:"display_name"`
	Email       *string `json:"email"`
	Role        string  `json:"role"`
	HasAccount  bool    `json:"has_account"`
}

func toParticipantJSON(p domain.Participant) participantJSON {
	out := participantJSON{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		DisplayName: p.DisplayName,
		Role:        string(p.Role),
		HasAccount:  p.HasAccount(),
	}
	if p.Email != "" {
		email := p.Email
		out.Email = &email
	}
	return out
}

type inviteJSON struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
}

// handleGetInvite: GET /api/invites/{code}. Public, so the join page can say
// "Join Dead Air" before the visitor has logged in. It reveals only the name,
// and only to someone who already holds the link.
func handleGetInvite(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project, err := deps.Projects.ProjectByInviteCode(r.Context(), r.PathValue("code"))
		if errors.Is(err, store.ErrNotFound) {
			writeInviteNotFound(w)
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "get invite", err)
			return
		}
		writeJSON(w, http.StatusOK, inviteJSON{ProjectID: project.ID, ProjectName: project.Name})
	}
}

// handleJoinInvite: POST /api/invites/{code}/join. 201 when a participant was
// created, 200 when the caller was already in (or was linked to a row the
// producer made for them) — safe to call any number of times.
func handleJoinInvite(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Role string `json:"role"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		role, fieldErrs := domain.ValidateJoinRole(req.Role)
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		project, err := deps.Projects.ProjectByInviteCode(r.Context(), r.PathValue("code"))
		if errors.Is(err, store.ErrNotFound) {
			writeInviteNotFound(w)
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "join: find invite", err)
			return
		}

		user, ok := currentUser(w, r, deps)
		if !ok {
			return
		}

		participant, created, err := deps.Projects.JoinProject(r.Context(), user, project.ID, role)
		switch {
		case errors.Is(err, store.ErrNotFound):
			// The project was deleted between the lookup and the join.
			writeInviteNotFound(w)
			return
		case errors.Is(err, store.ErrParticipantEmailTaken):
			writeError(w, http.StatusConflict, "participant_email_taken",
				"someone in this project already uses your email address")
			return
		case err != nil:
			writeInternalError(w, deps.Logger, "join project", err)
			return
		}

		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(w, status, toParticipantJSON(participant))
	}
}

func writeInviteNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "invite_not_found", "this invite link does not work")
}
