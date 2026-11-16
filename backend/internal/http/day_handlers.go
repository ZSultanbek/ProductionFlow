package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// DayStore is the slice of store.DB the shooting-day handlers need.
type DayStore interface {
	CreateDay(ctx context.Context, projectID string, in domain.DayInput) (domain.ShootingDay, error)
	DaysForProject(ctx context.Context, projectID, from string, after *store.DayKeyset, limit int) ([]domain.ShootingDay, error)
	CallSheetForUser(ctx context.Context, userID, dayID string) (domain.CallSheet, error)
	TodayForUser(ctx context.Context, userID, date string) (string, []domain.AgendaDay, error)
}

// --- wire shapes (docs/api.md, "Shared shapes" and "Shooting days") ---

type dayJSON struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	ShootDate  string `json:"shoot_date"`
	CallTime   string `json:"call_time"`
	Location   string `json:"location"`
	SceneCount int    `json:"scene_count"`
	UpdatedAt  string `json:"updated_at"`
}

func toDayJSON(d domain.ShootingDay) dayJSON {
	return dayJSON{
		ID:         d.ID,
		ProjectID:  d.ProjectID,
		ShootDate:  d.ShootDate,
		CallTime:   d.CallTime,
		Location:   d.Location,
		SceneCount: d.SceneCount,
		UpdatedAt:  formatInstant(d.UpdatedAt),
	}
}

type participantRefJSON struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// toParticipantRefs never returns nil: an empty list must be [] in JSON, not
// null, so the client can iterate without a null check.
func toParticipantRefs(refs []domain.ParticipantRef) []participantRefJSON {
	out := make([]participantRefJSON, 0, len(refs))
	for _, r := range refs {
		out = append(out, participantRefJSON{ID: r.ID, DisplayName: r.DisplayName, Role: string(r.Role)})
	}
	return out
}

type sceneRefJSON struct {
	ID          string `json:"id"`
	Number      string `json:"number"`
	Description string `json:"description"`
}

type callSheetSceneJSON struct {
	sceneRefJSON
	Participants []participantRefJSON `json:"participants"`
}

type callSheetProjectJSON struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	MyRole string `json:"my_role"`
}

type callSheetJSON struct {
	ID          string               `json:"id"`
	Project     callSheetProjectJSON `json:"project"`
	ShootDate   string               `json:"shoot_date"`
	CallTime    string               `json:"call_time"`
	Location    string               `json:"location"`
	UpdatedAt   string               `json:"updated_at"`
	OnSetAllDay []participantRefJSON `json:"on_set_all_day"`
	Scenes      []callSheetSceneJSON `json:"scenes"`
}

func toCallSheetJSON(s domain.CallSheet) callSheetJSON {
	scenes := make([]callSheetSceneJSON, 0, len(s.Scenes))
	for _, sc := range s.Scenes {
		scenes = append(scenes, callSheetSceneJSON{
			sceneRefJSON: sceneRefJSON{ID: sc.ID, Number: sc.Number, Description: sc.Description},
			Participants: toParticipantRefs(sc.Participants),
		})
	}
	return callSheetJSON{
		ID:          s.Day.ID,
		Project:     callSheetProjectJSON{ID: s.Day.ProjectID, Name: s.ProjectName, MyRole: string(s.MyRole)},
		ShootDate:   s.Day.ShootDate,
		CallTime:    s.Day.CallTime,
		Location:    s.Day.Location,
		UpdatedAt:   formatInstant(s.Day.UpdatedAt),
		OnSetAllDay: toParticipantRefs(s.OnSetAllDay),
		Scenes:      scenes,
	}
}

func dayCursorOf(d domain.ShootingDay) any {
	return dayCursor{ShootDate: d.ShootDate, CallTime: d.CallTime, ID: d.ID}
}

// --- handlers ---

// handleListDays: GET /api/projects/{project_id}/days — any member, in shooting
// order. ?from=YYYY-MM-DD skips days before that date.
func handleListDays(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, ok := memberOf(w, r, deps, r.PathValue("project_id"))
		if !ok {
			return
		}

		cursor, limit, fieldErrs := parsePage[dayCursor](r)
		from := r.URL.Query().Get("from")
		if from != "" && !domain.IsDate(from) {
			fieldErrs["from"] = "must be a real date, YYYY-MM-DD"
		}
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}
		var after *store.DayKeyset
		if cursor != nil {
			after = &store.DayKeyset{ShootDate: cursor.ShootDate, CallTime: cursor.CallTime, ID: cursor.ID}
		}

		days, err := deps.Days.DaysForProject(r.Context(), m.Project.ID, from, after, limit+1)
		if err != nil {
			writeInternalError(w, deps.Logger, "list days", err)
			return
		}

		days, next := page(days, limit, dayCursorOf)
		data := make([]dayJSON, 0, len(days))
		for _, d := range days {
			data = append(data, toDayJSON(d))
		}
		writeJSON(w, http.StatusOK, listResponse[dayJSON]{Data: data, NextCursor: next})
	}
}

// handleCreateDay: POST /api/projects/{project_id}/days — producer only.
//
// This is the only way a day gets its date, call time and location without a
// preview: nobody is attached to a day that does not exist yet, so there is
// nobody to affect. Changing them afterwards goes through /api/changes/apply.
func handleCreateDay(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Permission first, input second: someone who may not create days gets
		// 403 or 404 whatever they sent, not a review of their request body.
		m, ok := producerOf(w, r, deps, r.PathValue("project_id"))
		if !ok {
			return
		}

		var req struct {
			ShootDate string `json:"shoot_date"`
			CallTime  string `json:"call_time"`
			Location  string `json:"location"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		in, fieldErrs := domain.ValidateDayInput(req.ShootDate, req.CallTime, req.Location)
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		day, err := deps.Days.CreateDay(r.Context(), m.Project.ID, in)
		if err != nil {
			writeInternalError(w, deps.Logger, "create day", err)
			return
		}
		writeJSON(w, http.StatusCreated, toDayJSON(day))
	}
}

// handleGetDay: GET /api/days/{day_id} — the call sheet, for any member of the
// day's project.
func handleGetDay(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sheet, err := deps.Days.CallSheetForUser(r.Context(), userIDFrom(r.Context()), r.PathValue("day_id"))
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "day_not_found", "no such shooting day")
			return
		}
		if err != nil {
			writeInternalError(w, deps.Logger, "get call sheet", err)
			return
		}
		writeJSON(w, http.StatusOK, toCallSheetJSON(sheet))
	}
}
