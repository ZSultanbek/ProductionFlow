package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// See store_test.go for how to run these: they need TEST_DATABASE_URL.

// production is a project a few people into its prep: Madina produces,
// Daniyar is crew with an account, Aisha is cast with one.
type production struct {
	db      *DB
	madina  domain.User
	m       domain.Membership
	daniyar domain.Participant
	aisha   domain.Participant
}

func newProduction(t *testing.T) production {
	t.Helper()
	db := testDB(t)
	madina := newUser(t, db, "madina")
	m, err := db.CreateProject(context.Background(), madina, "Dead Air")
	if err != nil {
		t.Fatal(err)
	}
	return production{
		db:      db,
		madina:  madina,
		m:       m,
		daniyar: joinAs(t, db, newUser(t, db, "daniyar"), m.Project.ID, domain.RoleCrew),
		aisha:   joinAs(t, db, newUser(t, db, "aisha"), m.Project.ID, domain.RoleCast),
	}
}

func TestCreateParticipantByNameAndList(t *testing.T) {
	p := newProduction(t)
	ctx := context.Background()
	pid := p.m.Project.ID

	ruslan, err := p.db.CreateParticipant(ctx, pid, domain.ParticipantInput{DisplayName: "Ruslan", Role: domain.RoleCrew})
	if err != nil {
		t.Fatal(err)
	}
	if ruslan.HasAccount() || ruslan.Email != "" || ruslan.Role != domain.RoleCrew {
		t.Errorf("ruslan = %+v, want crew with no account and no email", ruslan)
	}

	// Two people without an email are fine; the uniqueness is only on emails.
	if _, err := p.db.CreateParticipant(ctx, pid, domain.ParticipantInput{DisplayName: "Timur", Role: domain.RoleCast}); err != nil {
		t.Fatal(err)
	}

	people, err := p.db.ParticipantsForProject(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, x := range people {
		names = append(names, x.DisplayName)
	}
	// Producer, then crew by name, then cast by name, ignoring case. (Joined
	// people are named after their user, which newUser names in lower case.)
	want := []string{"madina", "daniyar", "Ruslan", "aisha", "Timur"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %v, want %v", names, want)
		}
	}
}

func TestCreateParticipantEmailIsUniqueInTheProject(t *testing.T) {
	p := newProduction(t)
	ctx := context.Background()

	_, err := p.db.CreateParticipant(ctx, p.m.Project.ID,
		domain.ParticipantInput{DisplayName: "Also Aisha", Role: domain.RoleCast, Email: p.aisha.Email})
	if !errors.Is(err, ErrParticipantEmailTaken) {
		t.Errorf("err = %v, want ErrParticipantEmailTaken", err)
	}
}

func TestCreateSceneAndListWithFilters(t *testing.T) {
	p := newProduction(t)
	ctx := context.Background()
	pid := p.m.Project.ID
	day := newDay(t, p.db, pid, "2026-11-02", "07:30", "Kok-Tobe")

	onDay, err := p.db.CreateScene(ctx, pid, domain.SceneInput{Number: "12A", Description: "Rooftop", ShootingDayID: day.ID})
	if err != nil {
		t.Fatal(err)
	}
	if onDay.ShootingDayID != day.ID || onDay.Participants == nil {
		t.Errorf("scene = %+v, want on the day with an empty (non-nil) people list", onDay)
	}
	if _, err := p.db.CreateScene(ctx, pid, domain.SceneInput{Number: "14"}); err != nil {
		t.Fatal(err)
	}

	if _, err := p.db.CreateScene(ctx, pid, domain.SceneInput{Number: "14"}); !errors.Is(err, ErrSceneNumberTaken) {
		t.Errorf("duplicate number: err = %v, want ErrSceneNumberTaken", err)
	}

	for name, tc := range map[string]struct {
		f    SceneFilter
		want []string
	}{
		"all":         {SceneFilter{}, []string{"12A", "14"}},
		"on the day":  {SceneFilter{ShootingDayID: day.ID}, []string{"12A"}},
		"unscheduled": {SceneFilter{UnscheduledOnly: true}, []string{"14"}},
	} {
		scenes, err := p.db.ScenesForProject(ctx, pid, tc.f)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, s := range scenes {
			got = append(got, s.Number)
		}
		if len(got) != len(tc.want) || (len(got) > 0 && got[0] != tc.want[0]) {
			t.Errorf("%s: numbers = %v, want %v", name, got, tc.want)
		}
	}
}

func TestSetSceneAndDayParticipantsReplace(t *testing.T) {
	p := newProduction(t)
	ctx := context.Background()
	pid := p.m.Project.ID
	day := newDay(t, p.db, pid, "2026-11-02", "07:30", "Kok-Tobe")
	sc, err := p.db.CreateScene(ctx, pid, domain.SceneInput{Number: "12A", ShootingDayID: day.ID})
	if err != nil {
		t.Fatal(err)
	}

	got, err := p.db.SetSceneParticipants(ctx, sc.ID, []string{p.aisha.ID, p.daniyar.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Participants) != 2 || got.Participants[0].ID != p.daniyar.ID {
		t.Errorf("participants = %+v, want daniyar (crew) then aisha (cast)", got.Participants)
	}

	// The list is replaced, not added to; empty clears it.
	got, err = p.db.SetSceneParticipants(ctx, sc.ID, []string{p.aisha.ID})
	if err != nil || len(got.Participants) != 1 {
		t.Errorf("after replace: %+v, err %v; want only aisha", got.Participants, err)
	}
	if got, _ = p.db.SetSceneParticipants(ctx, sc.ID, []string{}); len(got.Participants) != 0 {
		t.Errorf("after clear: %+v", got.Participants)
	}

	refs, err := p.db.SetDayParticipants(ctx, day.ID, []string{p.daniyar.ID})
	if err != nil || len(refs) != 1 || refs[0].ID != p.daniyar.ID {
		t.Errorf("on set all day = %+v, err %v", refs, err)
	}
	if refs, _ = p.db.SetDayParticipants(ctx, day.ID, []string{}); len(refs) != 0 {
		t.Errorf("after clear: %+v", refs)
	}

	if _, err := p.db.SetSceneParticipants(ctx, "not-a-uuid", nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("bad scene id: err = %v, want ErrNotFound", err)
	}
}

func TestUpdateSceneWritesOneChangeEventOnlyWhenSomethingChanged(t *testing.T) {
	p := newProduction(t)
	ctx := context.Background()
	pid := p.m.Project.ID
	day := newDay(t, p.db, pid, "2026-11-02", "07:30", "Kok-Tobe")
	sc, err := p.db.CreateScene(ctx, pid, domain.SceneInput{Number: "14", Description: "Kitchen"})
	if err != nil {
		t.Fatal(err)
	}

	moved, err := p.db.UpdateScene(ctx, p.m.ParticipantID, sc.ID,
		domain.ScenePatch{SetDay: true, ShootingDayID: day.ID, SetDescription: true, Description: "Kitchen"})
	if err != nil {
		t.Fatal(err)
	}
	if moved.ShootingDayID != day.ID || moved.Description != "Kitchen" {
		t.Errorf("moved = %+v", moved)
	}

	// The same patch again changes nothing and must not log anything.
	if _, err := p.db.UpdateScene(ctx, p.m.ParticipantID, sc.ID,
		domain.ScenePatch{SetDay: true, ShootingDayID: day.ID}); err != nil {
		t.Fatal(err)
	}

	rows, err := p.db.Pool.Query(ctx, `
		SELECT entity_type, actor_participant_id::text, diff
		FROM change_event WHERE entity_id = $1`, sc.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var events int
	for rows.Next() {
		events++
		var kind, actor string
		var raw []byte
		if err := rows.Scan(&kind, &actor, &raw); err != nil {
			t.Fatal(err)
		}
		var diff map[string]map[string]*string
		if err := json.Unmarshal(raw, &diff); err != nil {
			t.Fatal(err)
		}
		// Only the day changed: from null (unscheduled) to the day.
		if kind != "scene" || actor != p.m.ParticipantID || len(diff) != 1 ||
			diff["shooting_day_id"]["from"] != nil || *diff["shooting_day_id"]["to"] != day.ID {
			t.Errorf("event = %s by %s, diff %s", kind, actor, raw)
		}
	}
	if events != 1 {
		t.Errorf("%d change events, want exactly 1", events)
	}
}

// End to end through the database: what preview would show for a day.
func TestImpactInputFeedsTheImpactRule(t *testing.T) {
	p := newProduction(t)
	ctx := context.Background()
	pid := p.m.Project.ID
	day := newDay(t, p.db, pid, "2026-11-02", "07:30", "Kok-Tobe")
	ruslan, _ := p.db.CreateParticipant(ctx, pid, domain.ParticipantInput{DisplayName: "Ruslan", Role: domain.RoleCrew})

	a := newScene(t, p.db, pid, day.ID, "12A", p.aisha.ID)
	b := newScene(t, p.db, pid, day.ID, "13", p.aisha.ID, ruslan.ID)
	newScene(t, p.db, pid, "", "99", p.daniyar.ID) // unscheduled: never counts
	putOnSetAllDay(t, p.db, day.ID, p.daniyar.ID)

	in, err := p.db.ImpactInput(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	impact := domain.ComputeImpact(day, in)

	if len(impact.Scenes) != 2 || impact.Scenes[0].ID != a || impact.Scenes[1].ID != b {
		t.Errorf("scenes = %+v", impact.Scenes)
	}
	if len(impact.Participants) != 3 {
		t.Fatalf("participants = %+v, want daniyar, ruslan, aisha", impact.Participants)
	}
	d, r, ai := impact.Participants[0], impact.Participants[1], impact.Participants[2]
	if d.ID != p.daniyar.ID || !d.OnSetAllDay || len(d.SceneNumbers) != 0 {
		t.Errorf("daniyar = %+v, want on set all day only (scene 99 is unscheduled)", d)
	}
	if r.ID != ruslan.ID || r.HasAccount {
		t.Errorf("ruslan = %+v, want no account", r)
	}
	if ai.ID != p.aisha.ID || len(ai.SceneNumbers) != 2 {
		t.Errorf("aisha = %+v, want 12A and 13", ai)
	}
	if n := impact.WillNotify(p.m.ParticipantID); n != 2 {
		t.Errorf("will notify %d, want 2 (daniyar, aisha)", n)
	}
}

func TestDayAndSceneForUserArePrivate(t *testing.T) {
	p := newProduction(t)
	ctx := context.Background()
	day := newDay(t, p.db, p.m.Project.ID, "2026-11-02", "07:30", "Kok-Tobe")
	sc, _ := p.db.CreateScene(ctx, p.m.Project.ID, domain.SceneInput{Number: "1", ShootingDayID: day.ID})
	stranger := newUser(t, p.db, "stranger")

	if d, m, err := p.db.DayForUser(ctx, p.madina.ID, day.ID); err != nil || d.SceneCount != 1 || m.Role != domain.RoleProducer {
		t.Errorf("producer: day %+v, membership %+v, err %v", d, m, err)
	}
	if _, _, err := p.db.DayForUser(ctx, stranger.ID, day.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("stranger day: err = %v, want ErrNotFound", err)
	}
	if _, _, err := p.db.SceneForUser(ctx, stranger.ID, sc.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("stranger scene: err = %v, want ErrNotFound", err)
	}
	if _, _, err := p.db.SceneForUser(ctx, p.madina.ID, "garbage"); !errors.Is(err, ErrNotFound) {
		t.Errorf("garbage id: err = %v, want ErrNotFound", err)
	}
}

// Two producers putting people on the same day at once: both succeed, one
// after the other, and the last list wins whole — never a mix of the two.
func TestSetDayParticipantsConcurrently(t *testing.T) {
	p := newProduction(t)
	ctx := context.Background()
	day := newDay(t, p.db, p.m.Project.ID, "2026-11-02", "07:30", "Kok-Tobe")

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids := []string{p.aisha.ID}
			if i%2 == 0 {
				ids = []string{p.daniyar.ID, p.aisha.ID}
			}
			_, errs[i] = p.db.SetDayParticipants(ctx, day.ID, ids)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	sheet, err := p.db.CallSheetForUser(ctx, p.madina.ID, day.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(sheet.OnSetAllDay); n != 1 && n != 2 {
		t.Errorf("on set all day = %+v", sheet.OnSetAllDay)
	}
}
