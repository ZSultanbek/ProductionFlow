package domain

import (
	"strings"
	"testing"
)

func TestIsDate(t *testing.T) {
	for s, want := range map[string]bool{
		"2026-10-04":           true,
		"2028-02-29":           true,  // leap year
		"2026-02-29":           false, // not a leap year
		"2026-13-01":           false,
		"2026-10-4":            false, // must be zero-padded
		"04.10.2026":           false,
		"2026-10-04 ":          false,
		"":                     false,
		"2026-10-04T00:00:00Z": false,
	} {
		if got := IsDate(s); got != want {
			t.Errorf("IsDate(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestIsCallTime(t *testing.T) {
	for s, want := range map[string]bool{
		"07:30":    true,
		"00:00":    true,
		"23:59":    true,
		"24:00":    false,
		"7:30":     false, // must be zero-padded
		"07:60":    false,
		"07:30:00": false, // no seconds
		"7.30":     false,
		"":         false,
	} {
		if got := IsCallTime(s); got != want {
			t.Errorf("IsCallTime(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestValidateDayInputAcceptsAndTrims(t *testing.T) {
	in, errs := ValidateDayInput(" 2026-10-04 ", " 07:30 ", "  Kok-Tobe, upper lot  ")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if in.ShootDate != "2026-10-04" || in.CallTime != "07:30" || in.Location != "Kok-Tobe, upper lot" {
		t.Errorf("not trimmed: %+v", in)
	}
}

func TestValidateDayInputAllowsPastDatesAndEmptyLocation(t *testing.T) {
	if _, errs := ValidateDayInput("2019-01-01", "07:30", ""); len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidateDayInputReportsEveryField(t *testing.T) {
	_, errs := ValidateDayInput("tomorrow", "7am", strings.Repeat("x", 201))
	for _, f := range []string{"shoot_date", "call_time", "location"} {
		if errs[f] == "" {
			t.Errorf("no error for %s: %v", f, errs)
		}
	}

	_, errs = ValidateDayInput("", "", "")
	if errs["shoot_date"] != "is required" || errs["call_time"] != "is required" {
		t.Errorf("missing fields not reported as required: %v", errs)
	}
}

func TestValidateDayInputLocationCountsCharacters(t *testing.T) {
	// 200 Cyrillic letters are 400 bytes; the limit is in characters.
	if _, errs := ValidateDayInput("2026-10-04", "07:30", strings.Repeat("ы", 200)); len(errs) != 0 {
		t.Errorf("200 characters rejected: %v", errs)
	}
}
