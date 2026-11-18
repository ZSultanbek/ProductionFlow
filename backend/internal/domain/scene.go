package domain

import (
	"strings"
	"unicode/utf8"
)

// Scene is one scene of the script. ShootingDayID is "" for an unscheduled
// scene: it exists, it just is not on any day yet, and it never counts towards
// any day's impact.
type Scene struct {
	ID            string
	ProjectID     string
	ShootingDayID string
	Number        string
	Description   string
	Participants  []ParticipantRef
}

// SceneInput is the cleaned-up input for creating a scene.
type SceneInput struct {
	Number        string
	Description   string
	ShootingDayID string // "" = unscheduled
}

// ScenePatch is a partial edit of a scene. The Set* flags say which fields the
// request contained: "description absent" and "description set to empty" are
// different requests.
type ScenePatch struct {
	SetDay         bool
	ShootingDayID  string // with SetDay: "" unschedules the scene
	SetDescription bool
	Description    string
}

// Change is one field of a change event: what it was and what it became. A nil
// side is JSON null — a scene that was unscheduled has no day to come from.
//
// The json tags are here, in domain, on purpose: this is the format stored in
// change_event.diff and shown in the change feed, so it is part of the rules,
// not a detail of one handler.
type Change struct {
	From *string `json:"from"`
	To   *string `json:"to"`
}

// Diff holds only the fields that changed, keyed by their API name.
type Diff map[string]Change

func ptr(s string) *string { return &s }

// optional is nil for "", so "no day" becomes null in a diff.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

const (
	maxSceneNumberLen      = 20
	maxSceneDescriptionLen = 500
)

// ValidateSceneInput checks a new scene. Whether the day belongs to the same
// project is checked by the handler, which has to look the day up anyway.
func ValidateSceneInput(number, description, shootingDayID string) (SceneInput, FieldErrors) {
	errs := FieldErrors{}
	in := SceneInput{
		Number:        strings.TrimSpace(number),
		Description:   strings.TrimSpace(description),
		ShootingDayID: strings.TrimSpace(shootingDayID),
	}

	switch n := utf8.RuneCountInString(in.Number); {
	case n == 0:
		errs["number"] = "is required"
	case n > maxSceneNumberLen:
		errs["number"] = "must be at most 20 characters"
	}
	if msg := checkSceneDescription(in.Description); msg != "" {
		errs["description"] = msg
	}
	return in, errs
}

// CleanScenePatch trims what the patch sets and checks it.
func CleanScenePatch(p ScenePatch) (ScenePatch, FieldErrors) {
	errs := FieldErrors{}
	p.ShootingDayID = strings.TrimSpace(p.ShootingDayID)
	p.Description = strings.TrimSpace(p.Description)
	if p.SetDescription {
		if msg := checkSceneDescription(p.Description); msg != "" {
			errs["description"] = msg
		}
	}
	return p, errs
}

func checkSceneDescription(s string) string {
	if utf8.RuneCountInString(s) > maxSceneDescriptionLen {
		return "must be at most 500 characters"
	}
	return ""
}

// DiffScene compares a scene with an edit to it and returns only what would
// really change. Empty means the edit is a no-op, and then no change event is
// written (docs/api.md, PATCH /api/scenes).
func DiffScene(current Scene, p ScenePatch) Diff {
	diff := Diff{}
	if p.SetDay && p.ShootingDayID != current.ShootingDayID {
		diff["shooting_day_id"] = Change{From: optional(current.ShootingDayID), To: optional(p.ShootingDayID)}
	}
	if p.SetDescription && p.Description != current.Description {
		diff["description"] = Change{From: ptr(current.Description), To: ptr(p.Description)}
	}
	return diff
}
