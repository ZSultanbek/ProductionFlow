package http

import (
	"errors"
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// handleListParticipants: GET /api/projects/{project_id}/participants — any
// member. Everyone in the project, producers first, then crew, then cast.
func handleListParticipants(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, ok := memberOf(w, r, deps, r.PathValue("project_id"))
		if !ok {
			return
		}
		people, err := deps.Projects.ParticipantsForProject(r.Context(), m.Project.ID)
		if err != nil {
			writeInternalError(w, deps.Logger, "list participants", err)
			return
		}
		data := make([]participantJSON, 0, len(people))
		for _, p := range people {
			data = append(data, toParticipantJSON(p))
		}
		// A crew is 5–15 people: one page, always. The envelope is still the
		// list envelope, so the client reads every list the same way.
		writeJSON(w, http.StatusOK, listResponse[participantJSON]{Data: data})
	}
}

// handleCreateParticipant: POST /api/projects/{project_id}/participants —
// producer only. Adds someone by name who may never install the app; they still
// count in every impact preview.
func handleCreateParticipant(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, ok := producerOf(w, r, deps, r.PathValue("project_id"))
		if !ok {
			return
		}

		var req struct {
			DisplayName string  `json:"display_name"`
			Role        string  `json:"role"`
			Email       *string `json:"email"` // null and absent both mean "not known"
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		email := ""
		if req.Email != nil {
			email = *req.Email
		}
		in, fieldErrs := domain.ValidateParticipantInput(req.DisplayName, req.Role, email)
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		p, err := deps.Projects.CreateParticipant(r.Context(), m.Project.ID, in)
		if errors.Is(err, store.ErrParticipantEmailTaken) {
			writeError(w, http.StatusConflict, "participant_email_taken",
				"someone in this project already has that email")
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "create participant", err)
			return
		}
		writeJSON(w, http.StatusCreated, toParticipantJSON(p))
	}
}

// handleSetDayParticipants: PUT /api/days/{day_id}/participants — producer
// only. Replaces who is on set for the whole day.
func handleSetDayParticipants(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		day, m, ok := dayOf(w, r, deps, r.PathValue("day_id"), true)
		if !ok {
			return
		}
		ids, ok := participantIDsFrom(w, r, deps, m.Project.ID)
		if !ok {
			return
		}

		refs, err := deps.Days.SetDayParticipants(r.Context(), day.ID, ids)
		if err != nil {
			writeInternalError(w, deps.Logger, "set day participants", err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			OnSetAllDay []participantRefJSON `json:"on_set_all_day"`
		}{toParticipantRefs(refs)})
	}
}

// participantIDsFrom reads {"participant_ids": [...]} and checks that every id
// is someone in the project. An empty list is valid — it clears the list — but
// a missing one is not: "remove everybody" should never happen by accident.
// On false a response has been written.
func participantIDsFrom(w http.ResponseWriter, r *http.Request, deps Deps, projectID string) ([]string, bool) {
	var req struct {
		ParticipantIDs *[]string `json:"participant_ids"`
	}
	if !decodeJSON(w, r, &req) {
		return nil, false
	}
	if req.ParticipantIDs == nil {
		writeValidationError(w, domain.FieldErrors{"participant_ids": "is required; send [] to clear the list"})
		return nil, false
	}

	people, err := deps.Projects.ParticipantsForProject(r.Context(), projectID)
	if err != nil {
		writeInternalError(w, deps.Logger, "load participants", err)
		return nil, false
	}
	ids, fieldErrs := domain.CheckParticipantIDs(people, *req.ParticipantIDs)
	if len(fieldErrs) > 0 {
		writeValidationError(w, fieldErrs)
		return nil, false
	}
	return ids, true
}

// dayOf loads a day the caller can see, and their membership of its project.
// Not found and not a member are the same 404, as for projects. With
// producerOnly, a member who is not a producer gets 403.
// On false a response has been written.
func dayOf(w http.ResponseWriter, r *http.Request, deps Deps, dayID string, producerOnly bool) (domain.ShootingDay, domain.Membership, bool) {
	day, m, err := deps.Days.DayForUser(r.Context(), userIDFrom(r.Context()), dayID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "day_not_found", "no such shooting day")
		return domain.ShootingDay{}, domain.Membership{}, false
	}
	if err != nil {
		writeInternalError(w, deps.Logger, "load day", err)
		return domain.ShootingDay{}, domain.Membership{}, false
	}
	if producerOnly && m.Role != domain.RoleProducer {
		writeError(w, http.StatusForbidden, "forbidden", "only a producer of this project can do that")
		return domain.ShootingDay{}, domain.Membership{}, false
	}
	return day, m, true
}
