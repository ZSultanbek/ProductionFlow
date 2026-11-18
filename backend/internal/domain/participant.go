package domain

import (
	"sort"
	"strings"
)

// ParticipantInput is the cleaned-up input for adding a person by name.
type ParticipantInput struct {
	DisplayName string
	Role        Role
	Email       string // "" when the producer does not know it
}

// ValidateParticipantInput checks a person the producer adds by hand (US-06).
// The email is optional: plenty of cast are known only by name and phone. When
// it is given, it is what links the row to their account if they register and
// follow the invite later.
func ValidateParticipantInput(displayName, role, email string) (ParticipantInput, FieldErrors) {
	errs := FieldErrors{}
	in := ParticipantInput{
		DisplayName: strings.TrimSpace(displayName),
		Email:       strings.TrimSpace(email),
	}

	if msg := checkDisplayName(in.DisplayName); msg != "" {
		errs["display_name"] = msg
	}
	// Same rule as joining: producer is never handed out from a client.
	r, roleErrs := ValidateJoinRole(role)
	in.Role = r
	for k, v := range roleErrs {
		errs[k] = v
	}
	if in.Email != "" {
		if msg := checkEmail(in.Email); msg != "" {
			errs["email"] = msg
		}
	}
	return in, errs
}

// CheckParticipantIDs is the rule behind every "replace the people on X"
// request: each id must be a participant of the project. It returns the ids
// without duplicates, or the field error naming the ones that are not in it.
//
// people is the whole project; at 5–15 people that is cheaper and simpler than
// a query per id.
func CheckParticipantIDs(people []Participant, ids []string) ([]string, FieldErrors) {
	inProject := make(map[string]bool, len(people))
	for _, p := range people {
		inProject[p.ID] = true
	}

	seen := map[string]bool{}
	clean := []string{}
	var unknown []string
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		if !inProject[id] {
			unknown = append(unknown, id)
			continue
		}
		clean = append(clean, id)
	}

	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, FieldErrors{"participant_ids": "not in this project: " + strings.Join(unknown, ", ")}
	}
	return clean, nil
}
