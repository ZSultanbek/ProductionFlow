package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateSceneInput(t *testing.T) {
	in, errs := ValidateSceneInput(" 12A ", " Rooftop ", "")
	if len(errs) != 0 || in.Number != "12A" || in.Description != "Rooftop" || in.ShootingDayID != "" {
		t.Errorf("in = %+v, errs = %v", in, errs)
	}

	_, errs = ValidateSceneInput("  ", strings.Repeat("x", 501), "")
	if errs["number"] == "" || errs["description"] == "" {
		t.Errorf("missing number and long description not both reported: %v", errs)
	}

	if _, errs = ValidateSceneInput(strings.Repeat("1", 21), "", ""); errs["number"] == "" {
		t.Error("a 21-character number was accepted")
	}
}

func TestDiffSceneHoldsOnlyRealChanges(t *testing.T) {
	scene := Scene{ID: "sc-1", ShootingDayID: "day-mon", Number: "14", Description: "Kitchen"}

	// Unscheduling: the day becomes null.
	diff := DiffScene(scene, ScenePatch{SetDay: true, ShootingDayID: "", SetDescription: true, Description: "Kitchen"})
	want := Diff{"shooting_day_id": {From: s("day-mon"), To: nil}}
	if !reflect.DeepEqual(diff, want) {
		t.Errorf("diff = %v, want only shooting_day_id day-mon → null", diff)
	}

	// Sending what is already there is not a change.
	if diff := DiffScene(scene, ScenePatch{SetDay: true, ShootingDayID: "day-mon"}); len(diff) != 0 {
		t.Errorf("no-op patch gave diff %v", diff)
	}

	// Fields not in the patch are not touched, even if empty.
	if diff := DiffScene(scene, ScenePatch{}); len(diff) != 0 {
		t.Errorf("empty patch gave diff %v", diff)
	}
}

func TestValidateParticipantInput(t *testing.T) {
	in, errs := ValidateParticipantInput(" Ruslan ", "crew", "")
	if len(errs) != 0 || in.DisplayName != "Ruslan" || in.Role != RoleCrew || in.Email != "" {
		t.Errorf("in = %+v, errs = %v", in, errs)
	}

	_, errs = ValidateParticipantInput("", "producer", "not-an-email")
	if errs["display_name"] == "" || errs["role"] == "" || errs["email"] == "" {
		t.Errorf("want all three fields reported, got %v", errs)
	}
}

func TestCheckParticipantIDs(t *testing.T) {
	ids, errs := CheckParticipantIDs(everyone, []string{"p-aisha", "p-timur", "p-aisha"})
	if len(errs) != 0 || !reflect.DeepEqual(ids, []string{"p-aisha", "p-timur"}) {
		t.Errorf("ids = %v, errs = %v; want duplicates dropped", ids, errs)
	}

	_, errs = CheckParticipantIDs(everyone, []string{"p-aisha", "p-zz", "p-aa"})
	if errs["participant_ids"] != "not in this project: p-aa, p-zz" {
		t.Errorf("errs = %v", errs)
	}

	if ids, errs := CheckParticipantIDs(everyone, nil); len(errs) != 0 || ids == nil || len(ids) != 0 {
		t.Errorf("empty list: ids = %#v, errs = %v; want [] (clears the list)", ids, errs)
	}
}
