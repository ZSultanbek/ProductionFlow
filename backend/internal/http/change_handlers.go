package http

import (
	"errors"
	"net/http"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/auth"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// --- wire shapes (docs/api.md, "The impact view") ---

type viaJSON struct {
	Day          bool     `json:"day"`
	SceneNumbers []string `json:"scene_numbers"`
}

type affectedParticipantJSON struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	HasAccount  bool    `json:"has_account"`
	Via         viaJSON `json:"via"`
}

type previewJSON struct {
	PreviewToken string `json:"preview_token"`
	ExpiresAt    string `json:"expires_at"`
	Change       struct {
		Kind          string      `json:"kind"`
		ShootingDayID string      `json:"shooting_day_id"`
		Diff          domain.Diff `json:"diff"`
	} `json:"change"`
	Affected struct {
		Scenes       []sceneRefJSON            `json:"scenes"`
		Participants []affectedParticipantJSON `json:"participants"`
	} `json:"affected"`
	WillNotifyCount int `json:"will_notify_count"`
}

// changeRequest is the body of preview, and (with a token) of apply.
type changeRequest struct {
	Kind          string `json:"kind"`
	ShootingDayID string `json:"shooting_day_id"`
	// A field left out — or sent as null — stays as it is.
	To *struct {
		ShootDate *string `json:"shoot_date"`
		CallTime  *string `json:"call_time"`
		Location  *string `json:"location"`
	} `json:"to"`
}

// handlePreviewChange: POST /api/changes/preview — producer only. Shows who and
// what moving a day would affect, and writes nothing. The steps follow
// docs/architecture.md §4.4; the rule itself is domain.ComputeImpact.
func handlePreviewChange(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req changeRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		// The day decides whose project this is, so it has to be read before
		// the permission check — but nothing else in the body is looked at
		// until the caller is known to be its producer.
		if req.ShootingDayID == "" {
			writeValidationError(w, domain.FieldErrors{"shooting_day_id": "is required"})
			return
		}
		day, m, ok := dayOf(w, r, deps, req.ShootingDayID, true)
		if !ok {
			return
		}

		fieldErrs := domain.FieldErrors{}
		if req.Kind != domain.ChangeKindMoveDay {
			fieldErrs["kind"] = "must be move_shooting_day"
		}
		var move domain.DayMove
		if req.To == nil {
			fieldErrs["to"] = "is required"
		} else {
			var moveErrs domain.FieldErrors
			move, moveErrs = domain.ValidateDayMove(domain.DayMove{
				ShootDate: req.To.ShootDate, CallTime: req.To.CallTime, Location: req.To.Location,
			})
			for k, v := range moveErrs {
				fieldErrs[k] = v
			}
		}
		if len(fieldErrs) > 0 {
			writeValidationError(w, fieldErrs)
			return
		}

		diff, err := move.Diff(day)
		if errors.Is(err, domain.ErrNoChange) {
			writeError(w, http.StatusUnprocessableEntity, "no_change", "the day already has these values")
			return
		}

		in, err := deps.Scenes.ImpactInput(r.Context(), day)
		if err != nil {
			writeInternalError(w, deps.Logger, "load impact", err)
			return
		}
		impact := domain.ComputeImpact(day, in)

		token, expiresAt := deps.Previews.Sign(auth.PreviewClaims{
			Kind:         domain.ChangeKindMoveDay,
			DayID:        day.ID,
			MoveHash:     move.Fingerprint(),
			DayUpdatedAt: day.UpdatedAt.UnixMicro(),
			ImpactHash:   impact.Fingerprint(),
			UserID:       userIDFrom(r.Context()),
		})

		var out previewJSON
		out.PreviewToken = token
		out.ExpiresAt = formatInstant(expiresAt)
		out.Change.Kind = domain.ChangeKindMoveDay
		out.Change.ShootingDayID = day.ID
		out.Change.Diff = diff
		out.Affected.Scenes = make([]sceneRefJSON, 0, len(impact.Scenes))
		for _, s := range impact.Scenes {
			out.Affected.Scenes = append(out.Affected.Scenes, sceneRefJSON{ID: s.ID, Number: s.Number, Description: s.Description})
		}
		out.Affected.Participants = make([]affectedParticipantJSON, 0, len(impact.Participants))
		for _, p := range impact.Participants {
			out.Affected.Participants = append(out.Affected.Participants, affectedParticipantJSON{
				ID: p.ID, DisplayName: p.DisplayName, Role: string(p.Role), HasAccount: p.HasAccount,
				Via: viaJSON{Day: p.OnSetAllDay, SceneNumbers: p.SceneNumbers},
			})
		}
		out.WillNotifyCount = impact.WillNotify(m.ParticipantID)
		writeJSON(w, http.StatusOK, out)
	}
}
