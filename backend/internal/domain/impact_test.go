package domain

import (
	"errors"
	"reflect"
	"testing"
)

// The cast of these tests: one day, Dead Air's Monday on Kok-Tobe.
var monday = ShootingDay{ID: "day-mon", ShootDate: "2026-11-02", CallTime: "07:30", Location: "Kok-Tobe"}

var (
	madina  = Participant{ID: "p-madina", UserID: "u-madina", DisplayName: "Madina", Role: RoleProducer}
	daniyar = Participant{ID: "p-daniyar", UserID: "u-daniyar", DisplayName: "Daniyar", Role: RoleCrew}
	ruslan  = Participant{ID: "p-ruslan", DisplayName: "Ruslan", Role: RoleCrew} // added by name, no account
	aisha   = Participant{ID: "p-aisha", UserID: "u-aisha", DisplayName: "Aisha", Role: RoleCast}
	timur   = Participant{ID: "p-timur", UserID: "u-timur", DisplayName: "Timur", Role: RoleCast}
)

var everyone = []Participant{timur, aisha, ruslan, daniyar, madina}

func s(p string) *string { return &p }

func TestImpactScenesAndPeopleOfTheDay(t *testing.T) {
	in := ImpactInput{
		Scenes: []Scene{
			{ID: "sc-13", ShootingDayID: "day-mon", Number: "13", Description: "Insert: the note"},
			{ID: "sc-12a", ShootingDayID: "day-mon", Number: "12A", Description: "Rooftop"},
		},
		SceneLinks: []SceneLink{
			{SceneID: "sc-12a", ParticipantID: "p-aisha"},
			{SceneID: "sc-13", ParticipantID: "p-ruslan"},
		},
		DayLinks:     []string{"p-daniyar"},
		Participants: everyone,
	}

	got := ComputeImpact(monday, in)

	if len(got.Scenes) != 2 || got.Scenes[0].Number != "12A" || got.Scenes[1].Number != "13" {
		t.Fatalf("scenes = %+v, want 12A then 13", got.Scenes)
	}
	want := []AffectedParticipant{
		{ParticipantRef: ParticipantRef{ID: "p-daniyar", DisplayName: "Daniyar", Role: RoleCrew}, HasAccount: true, OnSetAllDay: true, SceneNumbers: []string{}},
		{ParticipantRef: ParticipantRef{ID: "p-ruslan", DisplayName: "Ruslan", Role: RoleCrew}, HasAccount: false, SceneNumbers: []string{"13"}},
		{ParticipantRef: ParticipantRef{ID: "p-aisha", DisplayName: "Aisha", Role: RoleCast}, HasAccount: true, SceneNumbers: []string{"12A"}},
	}
	if !reflect.DeepEqual(got.Participants, want) {
		t.Errorf("participants =\n%+v\nwant\n%+v", got.Participants, want)
	}
}

// US-07 scenario 3: someone in two scenes and on set all day appears once,
// with every reason.
func TestImpactListsEachPersonOnceWithEveryReason(t *testing.T) {
	in := ImpactInput{
		Scenes: []Scene{
			{ID: "sc-12a", ShootingDayID: "day-mon", Number: "12A"},
			{ID: "sc-12b", ShootingDayID: "day-mon", Number: "12B"},
		},
		SceneLinks: []SceneLink{
			{SceneID: "sc-12b", ParticipantID: "p-aisha"},
			{SceneID: "sc-12a", ParticipantID: "p-aisha"},
			{SceneID: "sc-12a", ParticipantID: "p-aisha"}, // a duplicate link changes nothing
		},
		DayLinks:     []string{"p-aisha"},
		Participants: everyone,
	}

	got := ComputeImpact(monday, in)

	if len(got.Participants) != 1 {
		t.Fatalf("got %d people, want Aisha once: %+v", len(got.Participants), got.Participants)
	}
	a := got.Participants[0]
	if !a.OnSetAllDay || !reflect.DeepEqual(a.SceneNumbers, []string{"12A", "12B"}) {
		t.Errorf("via = day:%v scenes:%v, want day:true scenes:[12A 12B]", a.OnSetAllDay, a.SceneNumbers)
	}
}

// US-06 scenario 3: unscheduled scenes, and scenes on other days, never count.
func TestImpactIgnoresScenesNotOnTheDay(t *testing.T) {
	in := ImpactInput{
		Scenes: []Scene{
			{ID: "sc-1", ShootingDayID: "day-mon", Number: "1"},
			{ID: "sc-2", ShootingDayID: "", Number: "2"},        // unscheduled
			{ID: "sc-3", ShootingDayID: "day-tue", Number: "3"}, // another day
		},
		SceneLinks: []SceneLink{
			{SceneID: "sc-2", ParticipantID: "p-timur"},
			{SceneID: "sc-3", ParticipantID: "p-aisha"},
		},
		Participants: everyone,
	}

	got := ComputeImpact(monday, in)

	if len(got.Scenes) != 1 || got.Scenes[0].ID != "sc-1" {
		t.Errorf("scenes = %+v, want only sc-1", got.Scenes)
	}
	if len(got.Participants) != 0 {
		t.Errorf("participants = %+v, want nobody", got.Participants)
	}
}

// US-07 scenario 2: an empty day is still a valid preview, with [] not nil.
func TestImpactOfAnEmptyDay(t *testing.T) {
	got := ComputeImpact(monday, ImpactInput{Participants: everyone})
	if got.Scenes == nil || got.Participants == nil || len(got.Scenes)+len(got.Participants) != 0 {
		t.Errorf("got %+v, want two empty non-nil lists", got)
	}
	if got.WillNotify("p-madina") != 0 {
		t.Error("an empty day notifies someone")
	}
}

func TestImpactOrdersByRoleThenName(t *testing.T) {
	in := ImpactInput{DayLinks: []string{"p-timur", "p-aisha", "p-ruslan", "p-daniyar", "p-madina"}, Participants: everyone}
	var names []string
	for _, p := range ComputeImpact(monday, in).Participants {
		names = append(names, p.DisplayName)
	}
	want := []string{"Madina", "Daniyar", "Ruslan", "Aisha", "Timur"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("order = %v, want %v", names, want)
	}
}

// "daniyar" typed in lower case still sorts before "Ruslan".
func TestImpactSortsNamesIgnoringCase(t *testing.T) {
	lower := Participant{ID: "p-lower", UserID: "u", DisplayName: "daniyar", Role: RoleCrew}
	in := ImpactInput{DayLinks: []string{"p-ruslan", "p-lower"}, Participants: []Participant{ruslan, lower}}
	got := ComputeImpact(monday, in).Participants
	if got[0].DisplayName != "daniyar" {
		t.Errorf("order = %s, %s; want daniyar first", got[0].DisplayName, got[1].DisplayName)
	}
}

func TestWillNotifySkipsTheActorAndPeopleWithoutAccounts(t *testing.T) {
	in := ImpactInput{DayLinks: []string{"p-madina", "p-daniyar", "p-ruslan", "p-aisha"}, Participants: everyone}
	impact := ComputeImpact(monday, in)
	// Madina makes the change; Ruslan has no account. Daniyar and Aisha remain.
	if got := impact.WillNotify("p-madina"); got != 2 {
		t.Errorf("WillNotify = %d, want 2", got)
	}
}

func TestImpactFingerprintChangesWhenSomeoneIsAdded(t *testing.T) {
	in := ImpactInput{
		Scenes:       []Scene{{ID: "sc-1", ShootingDayID: "day-mon", Number: "1"}},
		SceneLinks:   []SceneLink{{SceneID: "sc-1", ParticipantID: "p-aisha"}},
		Participants: everyone,
	}
	before := ComputeImpact(monday, in).Fingerprint()

	// The same data in a different order is the same impact.
	shuffled := in
	shuffled.Participants = []Participant{madina, aisha, timur, daniyar, ruslan}
	if ComputeImpact(monday, shuffled).Fingerprint() != before {
		t.Error("fingerprint depends on input order")
	}

	in.SceneLinks = append(in.SceneLinks, SceneLink{SceneID: "sc-1", ParticipantID: "p-timur"})
	if ComputeImpact(monday, in).Fingerprint() == before {
		t.Error("adding Timur to a scene on the day did not change the fingerprint")
	}
}

func TestValidateDayMove(t *testing.T) {
	if _, errs := ValidateDayMove(DayMove{}); errs["to"] == "" {
		t.Error("an empty move was accepted")
	}

	move, errs := ValidateDayMove(DayMove{ShootDate: s(" 2026-11-04 "), Location: s("  Studio B ")})
	if len(errs) != 0 || *move.ShootDate != "2026-11-04" || *move.Location != "Studio B" || move.CallTime != nil {
		t.Errorf("move = %+v, errs = %v", move, errs)
	}

	_, errs = ValidateDayMove(DayMove{ShootDate: s("2026-02-30"), CallTime: s("7:30")})
	if errs["to.shoot_date"] == "" || errs["to.call_time"] == "" {
		t.Errorf("bad date and time not both reported: %v", errs)
	}
}

func TestDayMoveDiffHoldsOnlyWhatChanges(t *testing.T) {
	diff, err := DayMove{ShootDate: s("2026-11-04"), CallTime: s("07:30")}.Diff(monday)
	if err != nil {
		t.Fatal(err)
	}
	want := Diff{"shoot_date": {From: s("2026-11-02"), To: s("2026-11-04")}}
	if !reflect.DeepEqual(diff, want) {
		t.Errorf("diff = %v, want only shoot_date (call time is the same)", diff)
	}

	if _, err := (DayMove{CallTime: s("07:30"), Location: s("Kok-Tobe")}).Diff(monday); !errors.Is(err, ErrNoChange) {
		t.Errorf("a move to the same values: err = %v, want ErrNoChange", err)
	}
}

func TestDayMoveFingerprintTellsAbsentFromEmpty(t *testing.T) {
	a := DayMove{CallTime: s("08:00")}.Fingerprint()
	b := DayMove{CallTime: s("08:00"), Location: s("")}.Fingerprint()
	if a == b {
		t.Error("leaving location out and clearing it have the same fingerprint")
	}
	if a != (DayMove{CallTime: s("08:00")}).Fingerprint() {
		t.Error("fingerprint is not stable")
	}
}
