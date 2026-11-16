package domain

import (
	"strings"
	"testing"
)

func TestValidateRegistrationAcceptsAndTrims(t *testing.T) {
	reg, errs := ValidateRegistration("  madina@example.kz ", " correct horse", "  Madina  ")

	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if reg.Email != "madina@example.kz" {
		t.Errorf("email = %q, want it trimmed", reg.Email)
	}
	if reg.DisplayName != "Madina" {
		t.Errorf("display_name = %q, want it trimmed", reg.DisplayName)
	}
	// A leading space is part of the password; trimming it would lock people out.
	if reg.Password != " correct horse" {
		t.Errorf("password = %q, must not be trimmed", reg.Password)
	}
}

func TestValidateRegistrationReportsEveryField(t *testing.T) {
	_, errs := ValidateRegistration("not-an-email", "short", "   ")

	for _, field := range []string{"email", "password", "display_name"} {
		if errs[field] == "" {
			t.Errorf("no error for %s; got %v", field, errs)
		}
	}
}

func TestValidateRegistrationEmailRules(t *testing.T) {
	for _, tc := range []struct {
		email string
		ok    bool
	}{
		{"a@b", true},
		{"aisha@mail.kz", true},
		{"", false},
		{"@mail.kz", false},
		{"aisha@", false},
		{"aisha mail.kz", false},
		{"ai sha@mail.kz", false},
		{strings.Repeat("a", 250) + "@b.kz", false},
	} {
		_, errs := ValidateRegistration(tc.email, "correct horse", "Aisha")
		if got := errs["email"] == ""; got != tc.ok {
			t.Errorf("email %q: valid = %v, want %v (%v)", tc.email, got, tc.ok, errs)
		}
	}
}

func TestValidateRegistrationPasswordBounds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		ok       bool
	}{
		{"7 bytes", strings.Repeat("x", 7), false},
		{"8 bytes", strings.Repeat("x", 8), true},
		{"72 bytes", strings.Repeat("x", 72), true},
		// bcrypt silently ignores everything past 72 bytes; we reject instead.
		{"73 bytes", strings.Repeat("x", 73), false},
		// 4 Cyrillic letters are 8 bytes in UTF-8. The rule is about bytes.
		{"8 bytes of Cyrillic", "пароль12", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, errs := ValidateRegistration("a@b", tc.password, "Aisha")
			if got := errs["password"] == ""; got != tc.ok {
				t.Errorf("valid = %v, want %v (%v)", got, tc.ok, errs)
			}
		})
	}
}

func TestValidateRegistrationDisplayNameCountsCharactersNotBytes(t *testing.T) {
	// 80 Kazakh letters are 160 bytes; the limit is 80 characters, so this passes.
	name := strings.Repeat("ә", 80)
	if _, errs := ValidateRegistration("a@b", "correct horse", name); errs["display_name"] != "" {
		t.Errorf("80 characters rejected: %v", errs)
	}
	if _, errs := ValidateRegistration("a@b", "correct horse", name+"ә"); errs["display_name"] == "" {
		t.Error("81 characters accepted")
	}
}

func TestValidateLoginOnlyRequiresPresence(t *testing.T) {
	email, errs := ValidateLogin(" madina@example.kz ", "x")
	if len(errs) != 0 {
		t.Fatalf("a short password must not be rejected at login: %v", errs)
	}
	if email != "madina@example.kz" {
		t.Errorf("email = %q, want it trimmed", email)
	}

	_, errs = ValidateLogin("", "")
	if errs["email"] == "" || errs["password"] == "" {
		t.Errorf("missing fields not reported: %v", errs)
	}
}
