package domain

import (
	"strings"
	"testing"
)

func TestValidateProjectName(t *testing.T) {
	if name, errs := ValidateProjectName("  Dead Air  "); len(errs) != 0 || name != "Dead Air" {
		t.Errorf("got %q, %v; want trimmed and valid", name, errs)
	}
	if _, errs := ValidateProjectName("   "); errs["name"] == "" {
		t.Error("blank name accepted")
	}
	// The limit is characters: 120 Kazakh letters are 240 bytes and still fine.
	if _, errs := ValidateProjectName(strings.Repeat("ұ", 120)); len(errs) != 0 {
		t.Errorf("120 characters rejected: %v", errs)
	}
	if _, errs := ValidateProjectName(strings.Repeat("ұ", 121)); errs["name"] == "" {
		t.Error("121 characters accepted")
	}
}

func TestValidateJoinRole(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Role
		ok   bool
	}{
		{"crew", RoleCrew, true},
		{"cast", RoleCast, true},
		{" cast ", RoleCast, true},
		{"producer", "", false}, // never self-assigned
		{"", "", false},
		{"Crew", "", false}, // the contract's values are lower-case
		{"director", "", false},
	} {
		got, errs := ValidateJoinRole(tc.in)
		if ok := len(errs) == 0; ok != tc.ok || got != tc.want {
			t.Errorf("ValidateJoinRole(%q) = %q, %v; want %q, ok=%v", tc.in, got, errs, tc.want, tc.ok)
		}
	}
}

func TestDecideJoin(t *testing.T) {
	producer := Participant{ID: "p-prod", UserID: "u-madina", Email: "madina@mail.kz", Role: RoleProducer}
	aishaByName := Participant{ID: "p-aisha", Email: "Aisha@Mail.kz", Role: RoleCast} // added by the producer, no account
	ruslanNoEmail := Participant{ID: "p-ruslan", Role: RoleCrew}                      // added by name only
	people := []Participant{producer, aishaByName, ruslanNoEmail}

	for _, tc := range []struct {
		name          string
		userID, email string
		want          JoinOutcome
		wantID        string
	}{
		{"the creator follows their own link", "u-madina", "madina@mail.kz", JoinAlreadyMember, "p-prod"},
		{"pre-added person, email differs only in case", "u-aisha", "aisha@mail.kz", JoinLinkExisting, "p-aisha"},
		{"someone new", "u-daniyar", "daniyar@mail.kz", JoinCreate, ""},
		// A participant with no email must never be matched to anyone.
		{"empty email matches nobody", "u-x", "", JoinCreate, ""},
		{"email owned by another account", "u-impostor", "madina@mail.kz", JoinEmailConflict, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, id := DecideJoin(people, tc.userID, tc.email)
			if got != tc.want || id != tc.wantID {
				t.Errorf("got (%v, %q), want (%v, %q)", got, id, tc.want, tc.wantID)
			}
		})
	}
}

// Membership by account wins over a matching email: an existing member who
// shares an email with a pre-added row is already in, not linked twice.
func TestDecideJoinMembershipBeatsEmailMatch(t *testing.T) {
	people := []Participant{
		{ID: "p-byname", Email: "aisha@mail.kz"},
		{ID: "p-member", UserID: "u-aisha", Email: "aisha@mail.kz"},
	}
	if got, id := DecideJoin(people, "u-aisha", "aisha@mail.kz"); got != JoinAlreadyMember || id != "p-member" {
		t.Errorf("got (%v, %q), want already-member p-member", got, id)
	}
}

func TestHasAccount(t *testing.T) {
	if (Participant{UserID: "u"}).HasAccount() != true || (Participant{}).HasAccount() != false {
		t.Error("HasAccount wrong")
	}
}
