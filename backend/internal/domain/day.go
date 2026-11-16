package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// ShootingDay is the central entity: everything the crew coordinates hangs off
// a day.
//
// ShootDate and CallTime are strings on purpose. A call time is a wall-clock
// time on a date — "07:30 on the 4th" — not an instant, and keeping them out of
// time.Time means nothing can ever shift them by a time zone (CLAUDE.md).
type ShootingDay struct {
	ID         string
	ProjectID  string
	ShootDate  string // YYYY-MM-DD
	CallTime   string // HH:MM, 24-hour
	Location   string
	SceneCount int
	UpdatedAt  time.Time
}

// DayInput is the cleaned-up input for creating a shooting day.
type DayInput struct {
	ShootDate string
	CallTime  string
	Location  string
}

// ParticipantRef is the short form of a participant shown inside other things.
type ParticipantRef struct {
	ID          string
	DisplayName string
	Role        Role
}

// SceneRef is the short form of a scene.
type SceneRef struct {
	ID          string
	Number      string
	Description string
}

// CallSheetScene is a scene on a call sheet with the people needed in it.
type CallSheetScene struct {
	SceneRef
	Participants []ParticipantRef
}

// CallSheet is one shooting day as the whole crew sees it.
type CallSheet struct {
	Day         ShootingDay
	ProjectName string
	MyRole      Role // the role of whoever asked, in this day's project
	OnSetAllDay []ParticipantRef
	Scenes      []CallSheetScene
}

// AgendaDay is a shooting day as seen by one person: which of its scenes they
// are in, and whether they are on set for the whole day.
type AgendaDay struct {
	Day         ShootingDay
	ProjectName string
	MyScenes    []SceneRef
	OnSetAllDay bool
}

const maxLocationLen = 200

// IsDate reports whether s is a real calendar date written as YYYY-MM-DD.
// The length check matters: time.Parse alone would also accept "2026-10-4".
func IsDate(s string) bool {
	if len(s) != len("2006-01-02") {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// IsCallTime reports whether s is a 24-hour time written as HH:MM.
func IsCallTime(s string) bool {
	if len(s) != len("15:04") {
		return false
	}
	_, err := time.Parse("15:04", s)
	return err == nil
}

// ValidateDayInput checks the three fields of a new shooting day, reporting
// every problem at once. Past dates are allowed: crews log days after the fact.
func ValidateDayInput(shootDate, callTime, location string) (DayInput, FieldErrors) {
	errs := FieldErrors{}
	in := DayInput{
		ShootDate: strings.TrimSpace(shootDate),
		CallTime:  strings.TrimSpace(callTime),
		Location:  strings.TrimSpace(location),
	}

	switch {
	case in.ShootDate == "":
		errs["shoot_date"] = "is required"
	case !IsDate(in.ShootDate):
		errs["shoot_date"] = "must be a real date, YYYY-MM-DD"
	}

	switch {
	case in.CallTime == "":
		errs["call_time"] = "is required"
	case !IsCallTime(in.CallTime):
		errs["call_time"] = "must be HH:MM, 24-hour"
	}

	if utf8.RuneCountInString(in.Location) > maxLocationLen {
		errs["location"] = "must be at most 200 characters"
	}

	return in, errs
}
