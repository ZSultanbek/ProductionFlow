package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// SceneStore is the slice of store.DB the scene and impact handlers need.
type SceneStore interface {
	ScenesForProject(ctx context.Context, projectID string, f store.SceneFilter) ([]domain.Scene, error)
	CreateScene(ctx context.Context, projectID string, in domain.SceneInput) (domain.Scene, error)
	SceneForUser(ctx context.Context, userID, sceneID string) (domain.Scene, domain.Membership, error)
	UpdateScene(ctx context.Context, actorParticipantID, sceneID string, p domain.ScenePatch) (domain.Scene, error)
	SetSceneParticipants(ctx context.Context, sceneID string, participantIDs []string) (domain.Scene, error)
	ImpactInput(ctx context.Context, day domain.ShootingDay) (domain.ImpactInput, error)
}

// --- wire shapes (docs/api.md, "Shared shapes") ---

type sceneJSON struct {
	ID            string               `json:"id"`
	ProjectID     string               `json:"project_id"`
	ShootingDayID *string              `json:"shooting_day_id"` // null = unscheduled
	Number        string               `json:"number"`
	Description   string               `json:"description"`
	Participants  []participantRefJSON `json:"participants"`
}

func toSceneJSON(s domain.Scene) sceneJSON {
	out := sceneJSON{
		ID:           s.ID,
		ProjectID:    s.ProjectID,
		Number:       s.Number,
		Description:  s.Description,
		Participants: toParticipantRefs(s.Participants),
	}
	if s.ShootingDayID != "" {
		day := s.ShootingDayID
		out.ShootingDayID = &day
	}
	return out
}

// --- handlers ---

// handleListScenes: GET /api/projects/{project_id}/scenes — any member.
// ?shooting_day_id= for one day's scenes, ?unscheduled=true for the ones on no
// day; not both.
func handleListScenes(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, ok := memberOf(w, r, deps, r.PathValue("project_id"))
		if !ok {
			return
		}

		q := r.URL.Query()
		filter := store.SceneFilter{ShootingDayID: q.Get("shooting_day_id")}
		fieldErrs := domain.FieldErrors{}
		switch q.Get("unscheduled") {
		case "", "false":
		case "true":
			filter.UnscheduledOnly = true
		default:
			fieldErrs["unscheduled"] = "must be true or false"
		}
		// Checked here because it is cast to uuid in SQL: a malformed one would
		// otherwise be a database error.
		if filter.ShootingDayID != "" && !isUUID(filter.ShootingDayID) {
			fieldErrs["shooting_day_id"] = "must be a shooting day id"
		}
		if filter.ShootingDayID != "" && filter.UnscheduledOnly {
			fieldErrs["unscheduled"] = "cannot be combined with shooting_day_id"
		}
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		scenes, err := deps.Scenes.ScenesForProject(r.Context(), m.Project.ID, filter)
		if err != nil {
			writeInternalError(w, deps.Logger, "list scenes", err)
			return
		}
		data := make([]sceneJSON, 0, len(scenes))
		for _, s := range scenes {
			data = append(data, toSceneJSON(s))
		}
		writeJSON(w, http.StatusOK, listResponse[sceneJSON]{Data: data})
	}
}

// handleCreateScene: POST /api/projects/{project_id}/scenes — producer only.
func handleCreateScene(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, ok := producerOf(w, r, deps, r.PathValue("project_id"))
		if !ok {
			return
		}

		var req struct {
			Number        string  `json:"number"`
			Description   string  `json:"description"`
			ShootingDayID *string `json:"shooting_day_id"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		dayID := ""
		if req.ShootingDayID != nil {
			dayID = *req.ShootingDayID
		}
		in, fieldErrs := domain.ValidateSceneInput(req.Number, req.Description, dayID)
		if len(fieldErrs) == 0 && !dayInProject(w, r, deps, m.Project.ID, in.ShootingDayID, fieldErrs) {
			return
		}
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		scene, err := deps.Scenes.CreateScene(r.Context(), m.Project.ID, in)
		if errors.Is(err, store.ErrSceneNumberTaken) {
			writeError(w, http.StatusConflict, "scene_number_taken", "this project already has a scene with that number")
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "create scene", err)
			return
		}
		writeJSON(w, http.StatusCreated, toSceneJSON(scene))
	}
}

// handleUpdateScene: PATCH /api/scenes/{scene_id} — producer only. Moves the
// scene to another day (or off every day) and/or rewrites its description.
//
// Moving a scene does not go through the impact preview and notifies nobody:
// that was a grooming decision, and docs/api.md says so. The change event the
// store writes is the record of it.
func handleUpdateScene(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scene, m, ok := sceneOf(w, r, deps, r.PathValue("scene_id"), true)
		if !ok {
			return
		}

		// RawMessage, not *string, to tell three cases apart: the field is
		// absent (leave it), null (for the day: unschedule) or a value.
		var req struct {
			ShootingDayID json.RawMessage `json:"shooting_day_id"`
			Description   json.RawMessage `json:"description"`
			Number        json.RawMessage `json:"number"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}

		var patch domain.ScenePatch
		fieldErrs := domain.FieldErrors{}
		if req.Number != nil {
			// A scene number is how the crew refers to a scene out loud.
			// Renumbering is a new scene, not an edit.
			fieldErrs["number"] = "cannot be changed"
		}
		if req.ShootingDayID != nil {
			patch.SetDay = true
			if string(req.ShootingDayID) != "null" && json.Unmarshal(req.ShootingDayID, &patch.ShootingDayID) != nil {
				fieldErrs["shooting_day_id"] = "must be a shooting day id or null"
			}
		}
		if req.Description != nil {
			patch.SetDescription = true
			if json.Unmarshal(req.Description, &patch.Description) != nil || string(req.Description) == "null" {
				fieldErrs["description"] = "must be a string"
			}
		}
		patch, cleanErrs := domain.CleanScenePatch(patch)
		for k, v := range cleanErrs {
			fieldErrs[k] = v
		}
		if len(fieldErrs) == 0 && patch.SetDay && !dayInProject(w, r, deps, scene.ProjectID, patch.ShootingDayID, fieldErrs) {
			return
		}
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		updated, err := deps.Scenes.UpdateScene(r.Context(), m.ParticipantID, scene.ID, patch)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "scene_not_found", "no such scene")
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "update scene", err)
			return
		}
		writeJSON(w, http.StatusOK, toSceneJSON(updated))
	}
}

// handleSetSceneParticipants: PUT /api/scenes/{scene_id}/participants —
// producer only. Replaces who is needed in the scene.
func handleSetSceneParticipants(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scene, m, ok := sceneOf(w, r, deps, r.PathValue("scene_id"), true)
		if !ok {
			return
		}
		ids, ok := participantIDsFrom(w, r, deps, m.Project.ID)
		if !ok {
			return
		}

		updated, err := deps.Scenes.SetSceneParticipants(r.Context(), scene.ID, ids)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "scene_not_found", "no such scene")
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "set scene participants", err)
			return
		}
		writeJSON(w, http.StatusOK, toSceneJSON(updated))
	}
}

// sceneOf is dayOf for scenes. On false a response has been written.
func sceneOf(w http.ResponseWriter, r *http.Request, deps Deps, sceneID string, producerOnly bool) (domain.Scene, domain.Membership, bool) {
	scene, m, err := deps.Scenes.SceneForUser(r.Context(), userIDFrom(r.Context()), sceneID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "scene_not_found", "no such scene")
		return domain.Scene{}, domain.Membership{}, false
	}
	if err != nil {
		writeInternalError(w, deps.Logger, "load scene", err)
		return domain.Scene{}, domain.Membership{}, false
	}
	if producerOnly && m.Role != domain.RoleProducer {
		writeError(w, http.StatusForbidden, "forbidden", "only a producer of this project can do that")
		return domain.Scene{}, domain.Membership{}, false
	}
	return scene, m, true
}

// dayInProject checks that dayID ("" = no day) is a day of projectID, adding
// to fieldErrs if not. A day in another project — even one of the caller's own
// — is a validation error, not a 404: the scene is what the request is about.
// It returns false only when it has written a 500.
func dayInProject(w http.ResponseWriter, r *http.Request, deps Deps, projectID, dayID string, fieldErrs domain.FieldErrors) bool {
	if dayID == "" {
		return true
	}
	const msg = "is not a shooting day of this project"
	if !isUUID(dayID) {
		fieldErrs["shooting_day_id"] = msg
		return true
	}
	day, _, err := deps.Days.DayForUser(r.Context(), userIDFrom(r.Context()), dayID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		fieldErrs["shooting_day_id"] = msg
	case err != nil:
		writeInternalError(w, deps.Logger, "check scene day", err)
		return false
	case day.ProjectID != projectID:
		fieldErrs["shooting_day_id"] = msg
	}
	return true
}
