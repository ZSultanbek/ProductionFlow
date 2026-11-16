package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Role is what a participant does on one production. It belongs to the
// participant, not the user: the same person can produce one film and hold a
// boom on another.
type Role string

const (
	RoleProducer Role = "producer"
	RoleCrew     Role = "crew"
	RoleCast     Role = "cast"
)

// Project is one film production.
type Project struct {
	ID         string
	Name       string
	InviteCode string
	CreatedAt  time.Time
}

// Participant is a person in a project. UserID and Email are empty for someone
// the producer added by name who never registered — they still appear on call
// sheets and in impact lists, they just cannot log in or be notified.
type Participant struct {
	ID          string
	ProjectID   string
	UserID      string
	DisplayName string
	Email       string
	Role        Role
}

// HasAccount reports whether this participant can log in.
func (p Participant) HasAccount() bool { return p.UserID != "" }

// Membership is a project as seen by one user: the project plus who they are in it.
type Membership struct {
	Project       Project
	ParticipantID string
	Role          Role
}

const maxProjectNameLen = 120

// ValidateProjectName trims the name and checks its length in characters.
func ValidateProjectName(name string) (string, FieldErrors) {
	name = strings.TrimSpace(name)
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		return name, FieldErrors{"name": "is required"}
	case n > maxProjectNameLen:
		return name, FieldErrors{"name": "must be at most 120 characters"}
	}
	return name, nil
}

// ValidateJoinRole accepts only the roles a person may give themselves.
// Producer is the creator's role and is never self-assigned through an invite.
func ValidateJoinRole(role string) (Role, FieldErrors) {
	switch r := Role(strings.TrimSpace(role)); r {
	case RoleCrew, RoleCast:
		return r, nil
	case "":
		return "", FieldErrors{"role": "is required"}
	default:
		return "", FieldErrors{"role": "must be crew or cast"}
	}
}

// JoinOutcome is what joining a project through its invite link should do.
type JoinOutcome int

const (
	// JoinAlreadyMember: the user already has a participant row here. Change
	// nothing; the requested role is ignored.
	JoinAlreadyMember JoinOutcome = iota
	// JoinLinkExisting: the producer added this person by name with this email
	// before they had an account. Attach the account to that row and keep the
	// role the producer gave it, instead of creating a duplicate person.
	JoinLinkExisting
	// JoinCreate: a new participant row with the role the user picked.
	JoinCreate
	// JoinEmailConflict: a participant with this email exists but belongs to a
	// different account. Should not happen while emails are unique per account,
	// but if it does, refuse rather than guess.
	JoinEmailConflict
)

// DecideJoin picks the outcome for a user joining a project, given everyone
// already in it. Pure, so every branch is tested without a database; the store
// loads the participants under a lock and carries out what this returns.
//
// The participant id is set for JoinAlreadyMember and JoinLinkExisting.
func DecideJoin(participants []Participant, userID, email string) (JoinOutcome, string) {
	for _, p := range participants {
		if p.UserID == userID {
			return JoinAlreadyMember, p.ID
		}
	}
	for _, p := range participants {
		if p.Email == "" || !strings.EqualFold(p.Email, email) {
			continue
		}
		if p.UserID == "" {
			return JoinLinkExisting, p.ID
		}
		return JoinEmailConflict, ""
	}
	return JoinCreate, ""
}
