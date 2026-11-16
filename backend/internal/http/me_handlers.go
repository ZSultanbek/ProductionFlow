package http

import (
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// Endpoints under /api/me/: everything about the caller across all their
// projects. (GET /api/me itself lives with the auth handlers.)

type agendaProjectJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type agendaDayJSON struct {
	ID             string            `json:"id"`
	Project        agendaProjectJSON `json:"project"`
	ShootDate      string            `json:"shoot_date"`
	CallTime       string            `json:"call_time"`
	Location       string            `json:"location"`
	MyScenes       []sceneRefJSON    `json:"my_scenes"`
	IAmOnSetAllDay bool              `json:"i_am_on_set_all_day"`
}

func toAgendaDayJSON(a domain.AgendaDay) agendaDayJSON {
	scenes := make([]sceneRefJSON, 0, len(a.MyScenes))
	for _, s := range a.MyScenes {
		scenes = append(scenes, sceneRefJSON{ID: s.ID, Number: s.Number, Description: s.Description})
	}
	return agendaDayJSON{
		ID:             a.Day.ID,
		Project:        agendaProjectJSON{ID: a.Day.ProjectID, Name: a.ProjectName},
		ShootDate:      a.Day.ShootDate,
		CallTime:       a.Day.CallTime,
		Location:       a.Day.Location,
		MyScenes:       scenes,
		IAmOnSetAllDay: a.OnSetAllDay,
	}
}

type todayResponse struct {
	// Date is the date the days below are on: the requested one, or the next
	// date with any shooting. null when nothing is scheduled from here on.
	Date    *string         `json:"date"`
	IsToday bool            `json:"is_today"`
	Days    []agendaDayJSON `json:"days"`
}

// handleToday: GET /api/me/today?date=YYYY-MM-DD — the first screen of the
// app (US-05): today's call sheets, or the next shooting date if today is free.
//
// date is required and comes from the phone. The server deliberately has no
// opinion on what day it is where the user is standing.
func handleToday(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		switch {
		case date == "":
			writeValidationError(w, map[string]string{"date": "is required: your local date, YYYY-MM-DD"})
			return
		case !domain.IsDate(date):
			writeValidationError(w, map[string]string{"date": "must be a real date, YYYY-MM-DD"})
			return
		}

		found, days, err := deps.Days.TodayForUser(r.Context(), userIDFrom(r.Context()), date)
		if err != nil {
			writeInternalError(w, deps.Logger, "today", err)
			return
		}

		resp := todayResponse{Days: make([]agendaDayJSON, 0, len(days))}
		if found != "" {
			resp.Date = &found
			resp.IsToday = found == date
		}
		for _, a := range days {
			resp.Days = append(resp.Days, toAgendaDayJSON(a))
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
