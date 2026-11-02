package domain

import (
	"strings"
	"unicode/utf8"
)

// FieldErrors maps a JSON field name to a human-readable problem with it. It is
// what the API sends back as error.fields on a 422. Empty means valid.
type FieldErrors map[string]string

// Limits from the field rules table in docs/api.md. Change them there first.
const (
	maxEmailLen       = 254
	MinPasswordBytes  = 8
	MaxPasswordBytes  = 72 // bcrypt ignores everything past 72 bytes; reject rather than truncate
	maxDisplayNameLen = 80
)

// Registration is the cleaned-up input for creating an account.
type Registration struct {
	Email       string
	Password    string
	DisplayName string
}

// ValidateRegistration trims what should be trimmed and checks every field,
// reporting all problems at once rather than stopping at the first.
// The password is never trimmed: a leading space is part of the password.
func ValidateRegistration(email, password, displayName string) (Registration, FieldErrors) {
	errs := FieldErrors{}
	reg := Registration{
		Email:       strings.TrimSpace(email),
		Password:    password,
		DisplayName: strings.TrimSpace(displayName),
	}

	if msg := checkEmail(reg.Email); msg != "" {
		errs["email"] = msg
	}
	if msg := checkPassword(reg.Password); msg != "" {
		errs["password"] = msg
	}
	if msg := checkDisplayName(reg.DisplayName); msg != "" {
		errs["display_name"] = msg
	}

	return reg, errs
}

// ValidateLogin only checks that both fields are present. Login deliberately
// does not re-apply the registration rules: telling someone "password must be
// 8 bytes" on a login form leaks nothing useful and confuses people whose
// account predates a rule change.
func ValidateLogin(email, password string) (string, FieldErrors) {
	errs := FieldErrors{}
	email = strings.TrimSpace(email)

	if email == "" {
		errs["email"] = "is required"
	}
	if password == "" {
		errs["password"] = "is required"
	}

	return email, errs
}

func checkEmail(email string) string {
	switch {
	case email == "":
		return "is required"
	case len(email) > maxEmailLen:
		return "must be at most 254 characters"
	case strings.Index(email, "@") < 1 || strings.HasSuffix(email, "@"):
		return "must be a valid email address"
	case strings.ContainsAny(email, " \t\r\n"):
		return "must be a valid email address"
	}
	return ""
}

func checkPassword(password string) string {
	switch n := len(password); {
	case n < MinPasswordBytes:
		return "must be at least 8 characters"
	case n > MaxPasswordBytes:
		return "must be at most 72 bytes"
	}
	return ""
}

func checkDisplayName(name string) string {
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		return "is required"
	case n > maxDisplayNameLen:
		return "must be at most 80 characters"
	}
	return ""
}
