package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// ChangeKindMoveDay is the only kind of change in the MVP: a new date, call
// time and/or location for an existing shooting day.
const ChangeKindMoveDay = "move_shooting_day"

// ErrNoChange: every field of a move equals what the day already has.
var ErrNoChange = errors.New("domain: the change would change nothing")

// DayMove is where a shooting day is being moved to. A nil field is left as it
// is — the request did not mention it.
type DayMove struct {
	ShootDate *string
	CallTime  *string
	Location  *string
}

// ValidateDayMove trims and checks the "to" of a move. At least one field must
// be present, and each present one follows the same rules as creating a day.
func ValidateDayMove(to DayMove) (DayMove, FieldErrors) {
	errs := FieldErrors{}
	if to.ShootDate == nil && to.CallTime == nil && to.Location == nil {
		errs["to"] = "must contain at least one of shoot_date, call_time, location"
		return to, errs
	}

	clean := DayMove{}
	if to.ShootDate != nil {
		clean.ShootDate = ptr(strings.TrimSpace(*to.ShootDate))
		if !IsDate(*clean.ShootDate) {
			errs["to.shoot_date"] = "must be a real date, YYYY-MM-DD"
		}
	}
	if to.CallTime != nil {
		clean.CallTime = ptr(strings.TrimSpace(*to.CallTime))
		if !IsCallTime(*clean.CallTime) {
			errs["to.call_time"] = "must be HH:MM, 24-hour"
		}
	}
	if to.Location != nil {
		clean.Location = ptr(strings.TrimSpace(*to.Location))
		if utf8.RuneCountInString(*clean.Location) > maxLocationLen {
			errs["to.location"] = "must be at most 200 characters"
		}
	}
	return clean, errs
}

// Diff is what the move would change on day: only fields that differ.
// ErrNoChange if nothing would.
func (m DayMove) Diff(day ShootingDay) (Diff, error) {
	diff := Diff{}
	if m.ShootDate != nil && *m.ShootDate != day.ShootDate {
		diff["shoot_date"] = Change{From: ptr(day.ShootDate), To: ptr(*m.ShootDate)}
	}
	if m.CallTime != nil && *m.CallTime != day.CallTime {
		diff["call_time"] = Change{From: ptr(day.CallTime), To: ptr(*m.CallTime)}
	}
	if m.Location != nil && *m.Location != day.Location {
		diff["location"] = Change{From: ptr(day.Location), To: ptr(*m.Location)}
	}
	if len(diff) == 0 {
		return nil, ErrNoChange
	}
	return diff, nil
}

// Fingerprint identifies the move, so apply can tell whether it is being asked
// for the same change the preview was made for. %q keeps "absent" and "empty"
// apart and stops one field from running into the next.
func (m DayMove) Fingerprint() string {
	part := func(s *string) string {
		if s == nil {
			return "-"
		}
		return fmt.Sprintf("%q", *s)
	}
	return digest("move|" + part(m.ShootDate) + "|" + part(m.CallTime) + "|" + part(m.Location))
}

// SceneLink says a participant is needed in a scene (a scene_participant row).
type SceneLink struct {
	SceneID       string
	ParticipantID string
}

// ImpactInput is everything the impact rule looks at, loaded by the store.
type ImpactInput struct {
	// Scenes may include scenes that are not on the day, or not on any day;
	// the rule ignores them. The store only loads the day's own, but the rule
	// does not rely on that.
	Scenes     []Scene
	SceneLinks []SceneLink
	// DayLinks are the participant ids on set for the whole day.
	DayLinks     []string
	Participants []Participant
}

// AffectedParticipant is one person a change reaches, with every reason why.
type AffectedParticipant struct {
	ParticipantRef
	HasAccount   bool
	OnSetAllDay  bool     // via.day
	SceneNumbers []string // via.scene_numbers, in scene order
}

// Impact is who and what a change to a shooting day affects.
type Impact struct {
	Scenes       []SceneRef
	Participants []AffectedParticipant
}

// ComputeImpact is the rule the product exists for (docs/architecture.md §4.4):
//
//   - affected scenes are the scenes scheduled on the day;
//   - affected people are everyone in one of those scenes, plus everyone on set
//     for the whole day — each person once, with every reason listed;
//   - scenes are ordered by number (as bytes), people by role (producer, crew,
//     cast) and then name ignoring case, so the list reads like a call sheet and the fingerprint is
//     the same every time for the same data.
//
// A day with nothing on it gives empty (not nil) lists.
func ComputeImpact(day ShootingDay, in ImpactInput) Impact {
	impact := Impact{Scenes: []SceneRef{}, Participants: []AffectedParticipant{}}

	sceneNumber := map[string]string{}
	for _, s := range in.Scenes {
		if s.ShootingDayID != day.ID {
			continue
		}
		sceneNumber[s.ID] = s.Number
		impact.Scenes = append(impact.Scenes, SceneRef{ID: s.ID, Number: s.Number, Description: s.Description})
	}
	sort.Slice(impact.Scenes, func(i, j int) bool {
		a, b := impact.Scenes[i], impact.Scenes[j]
		if a.Number != b.Number {
			return a.Number < b.Number
		}
		return a.ID < b.ID
	})

	people := make(map[string]Participant, len(in.Participants))
	for _, p := range in.Participants {
		people[p.ID] = p
	}

	affected := map[string]*AffectedParticipant{}
	reach := func(participantID string) *AffectedParticipant {
		p, ok := people[participantID]
		if !ok {
			return nil // a link to someone not loaded: nobody to show
		}
		a, ok := affected[p.ID]
		if !ok {
			a = &AffectedParticipant{
				ParticipantRef: ParticipantRef{ID: p.ID, DisplayName: p.DisplayName, Role: p.Role},
				HasAccount:     p.HasAccount(),
				SceneNumbers:   []string{},
			}
			affected[p.ID] = a
		}
		return a
	}

	for _, id := range in.DayLinks {
		if a := reach(id); a != nil {
			a.OnSetAllDay = true
		}
	}
	for _, link := range in.SceneLinks {
		number, onDay := sceneNumber[link.SceneID]
		if !onDay {
			continue
		}
		if a := reach(link.ParticipantID); a != nil && !contains(a.SceneNumbers, number) {
			a.SceneNumbers = append(a.SceneNumbers, number)
		}
	}

	for _, a := range affected {
		sort.Strings(a.SceneNumbers)
		impact.Participants = append(impact.Participants, *a)
	}
	sort.Slice(impact.Participants, func(i, j int) bool {
		a, b := impact.Participants[i], impact.Participants[j]
		if ra, rb := roleRank(a.Role), roleRank(b.Role); ra != rb {
			return ra < rb
		}
		// Case-insensitive first, as the call sheet sorts (store.peopleOrder).
		if la, lb := strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName); la != lb {
			return la < lb
		}
		if a.DisplayName != b.DisplayName {
			return a.DisplayName < b.DisplayName
		}
		return a.ID < b.ID
	})
	return impact
}

// WillNotify counts who would get a notification: affected people who have an
// account, except the person making the change — they already know.
func (i Impact) WillNotify(actorParticipantID string) int {
	n := 0
	for _, p := range i.Participants {
		if p.HasAccount && p.ID != actorParticipantID {
			n++
		}
	}
	return n
}

// Fingerprint identifies exactly this impact: which scenes, which people, and
// why each person is in it. Apply recomputes the impact and compares, so a
// person attached to the day after the preview makes the preview stale. Account
// status is included because it changes who gets notified.
func (i Impact) Fingerprint() string {
	var b strings.Builder
	for _, s := range i.Scenes {
		b.WriteString("s:" + s.ID + "\n")
	}
	for _, p := range i.Participants {
		fmt.Fprintf(&b, "p:%s:%t:%t:%s\n", p.ID, p.HasAccount, p.OnSetAllDay, strings.Join(p.SceneNumbers, ","))
	}
	return digest(b.String())
}

func roleRank(r Role) int {
	switch r {
	case RoleProducer:
		return 0
	case RoleCrew:
		return 1
	case RoleCast:
		return 2
	}
	return 3
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
