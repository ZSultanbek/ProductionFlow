package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// See store_test.go for how to run these: they need TEST_DATABASE_URL.

func newDay(t *testing.T, db *DB, projectID, date, callTime, location string) domain.ShootingDay {
	t.Helper()
	d, err := db.CreateDay(context.Background(), projectID,
		domain.DayInput{ShootDate: date, CallTime: callTime, Location: location})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// The scene endpoints are US-06; until then tests put scenes in with SQL.
func newScene(t *testing.T, db *DB, projectID, dayID, number string, participantIDs ...string) string {
	t.Helper()
	ctx := context.Background()
	var day *string
	if dayID != "" {
		day = &dayID
	}
	var id string
	if err := db.Pool.QueryRow(ctx, `
		INSERT INTO scene (project_id, shooting_day_id, number, description)
		VALUES ($1, $2, $3, 'Scene ' || $3)
		RETURNING id::text`, projectID, day, number).Scan(&id); err != nil {
		t.Fatal(err)
	}
	for _, pid := range participantIDs {
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO scene_participant (scene_id, participant_id) VALUES ($1, $2)`, id, pid); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

func putOnSetAllDay(t *testing.T, db *DB, dayID, participantID string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(),
		`INSERT INTO shooting_day_participant (shooting_day_id, participant_id) VALUES ($1, $2)`,
		dayID, participantID); err != nil {
		t.Fatal(err)
	}
}

func joinAs(t *testing.T, db *DB, u domain.User, projectID string, role domain.Role) domain.Participant {
	t.Helper()
	p, _, err := db.JoinProject(context.Background(), u, projectID, role)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A date and a call time are wall-clock values: they must come back exactly as
// they went in, whatever time zone the database session runs in. This test uses
// its own pool whose every connection is set to a zone 14 hours ahead of UTC.
func TestCreateDayKeepsWallClockValuesExactly(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	m, err := db.CreateProject(ctx, newUser(t, db, "madina"), "Dead Air")
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "Pacific/Kiritimati"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	farAway := &DB{Pool: pool}

	var zone string
	if err := pool.QueryRow(ctx, `SHOW TIME ZONE`).Scan(&zone); err != nil || zone != "Pacific/Kiritimati" {
		t.Fatalf("session time zone = %q, %v", zone, err)
	}

	d := newDay(t, farAway, m.Project.ID, "2026-10-04", "07:30", "Kok-Tobe, upper lot")
	if d.ShootDate != "2026-10-04" || d.CallTime != "07:30" || d.Location != "Kok-Tobe, upper lot" {
		t.Errorf("created day = %+v", d)
	}
	if d.ID == "" || d.ProjectID != m.Project.ID || d.UpdatedAt.IsZero() || d.SceneCount != 0 {
		t.Errorf("created day = %+v", d)
	}

	list, err := farAway.DaysForProject(ctx, m.Project.ID, "", nil, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if list[0].ShootDate != "2026-10-04" || list[0].CallTime != "07:30" {
		t.Errorf("listed day = %+v", list[0])
	}
}

func TestDaysForProjectOrdersFiltersAndCountsScenes(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	m, err := db.CreateProject(ctx, newUser(t, db, "madina"), "Dead Air")
	if err != nil {
		t.Fatal(err)
	}
	pid := m.Project.ID

	// Created out of order on purpose.
	late := newDay(t, db, pid, "2026-10-09", "13:00", "Studio B")
	early := newDay(t, db, pid, "2026-10-04", "07:30", "Kok-Tobe")
	sameDateLater := newDay(t, db, pid, "2026-10-04", "15:00", "Second unit")
	newScene(t, db, pid, early.ID, "1")
	newScene(t, db, pid, early.ID, "2")
	newScene(t, db, pid, "", "99") // unscheduled: counts for no day

	// Another project's day must never leak in.
	other, err := db.CreateProject(ctx, newUser(t, db, "other"), "Other film")
	if err != nil {
		t.Fatal(err)
	}
	newDay(t, db, other.Project.ID, "2026-10-05", "09:00", "Elsewhere")

	all, err := db.DaysForProject(ctx, pid, "", nil, 50)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{early.ID, sameDateLater.ID, late.ID}
	if len(all) != 3 {
		t.Fatalf("got %d days, want 3", len(all))
	}
	for i, id := range wantOrder {
		if all[i].ID != id {
			t.Errorf("position %d = %s (%s %s), want %s", i, all[i].ID, all[i].ShootDate, all[i].CallTime, id)
		}
	}
	if all[0].SceneCount != 2 || all[1].SceneCount != 0 {
		t.Errorf("scene counts = %d, %d; want 2, 0", all[0].SceneCount, all[1].SceneCount)
	}

	fromLater, err := db.DaysForProject(ctx, pid, "2026-10-05", nil, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(fromLater) != 1 || fromLater[0].ID != late.ID {
		t.Errorf("from=2026-10-05 gave %+v, want only the 9th", fromLater)
	}
}

// Two units on the same date at the same call time: paging must neither skip
// nor repeat either of them.
func TestDaysForProjectPagesThroughIdenticalDateAndTime(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	m, err := db.CreateProject(ctx, newUser(t, db, "madina"), "Two units")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		newDay(t, db, m.Project.ID, "2026-10-04", "07:30", fmt.Sprintf("Unit %d", i))
	}

	seen := map[string]bool{}
	var after *DayKeyset
	for page := 0; page < 10; page++ {
		list, err := db.DaysForProject(ctx, m.Project.ID, "", after, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range list {
			if seen[d.ID] {
				t.Fatalf("day %s returned twice", d.ID)
			}
			seen[d.ID] = true
		}
		if len(list) < 2 {
			break
		}
		last := list[len(list)-1]
		after = &DayKeyset{ShootDate: last.ShootDate, CallTime: last.CallTime, ID: last.ID}
	}
	if len(seen) != 5 {
		t.Errorf("saw %d days across pages, want 5", len(seen))
	}
}

func TestCallSheetForUser(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	madina := newUser(t, db, "madina")
	m, err := db.CreateProject(ctx, madina, "Dead Air")
	if err != nil {
		t.Fatal(err)
	}
	pid := m.Project.ID
	aisha := joinAs(t, db, newUser(t, db, "aisha"), pid, domain.RoleCast)
	daniyarUser := newUser(t, db, "daniyar")
	daniyar := joinAs(t, db, daniyarUser, pid, domain.RoleCrew)

	day := newDay(t, db, pid, "2026-10-04", "07:30", "Kok-Tobe")
	otherDay := newDay(t, db, pid, "2026-10-05", "06:00", "Tram stop")
	// Inserted as 12B then 12A, and cast before crew, to prove the ordering.
	newScene(t, db, pid, day.ID, "12B", aisha.ID, daniyar.ID)
	newScene(t, db, pid, day.ID, "12A", aisha.ID)
	newScene(t, db, pid, otherDay.ID, "20", aisha.ID) // another day: not on this sheet
	newScene(t, db, pid, "", "99", aisha.ID)          // unscheduled: not on any sheet
	putOnSetAllDay(t, db, day.ID, daniyar.ID)

	sheet, err := db.CallSheetForUser(ctx, daniyarUser.ID, day.ID)
	if err != nil {
		t.Fatal(err)
	}

	if sheet.Day.ShootDate != "2026-10-04" || sheet.Day.CallTime != "07:30" || sheet.Day.Location != "Kok-Tobe" {
		t.Errorf("day = %+v", sheet.Day)
	}
	if sheet.ProjectName != "Dead Air" || sheet.MyRole != domain.RoleCrew {
		t.Errorf("project %q, my role %q; want Dead Air, crew", sheet.ProjectName, sheet.MyRole)
	}
	if len(sheet.OnSetAllDay) != 1 || sheet.OnSetAllDay[0].ID != daniyar.ID {
		t.Errorf("on set all day = %+v, want only daniyar", sheet.OnSetAllDay)
	}

	if len(sheet.Scenes) != 2 || sheet.Scenes[0].Number != "12A" || sheet.Scenes[1].Number != "12B" {
		t.Fatalf("scenes = %+v, want 12A then 12B", sheet.Scenes)
	}
	if len(sheet.Scenes[0].Participants) != 1 || sheet.Scenes[0].Participants[0].ID != aisha.ID {
		t.Errorf("12A participants = %+v, want only aisha", sheet.Scenes[0].Participants)
	}
	// Crew is listed before cast.
	in12B := sheet.Scenes[1].Participants
	if len(in12B) != 2 || in12B[0].ID != daniyar.ID || in12B[1].ID != aisha.ID {
		t.Errorf("12B participants = %+v, want daniyar (crew) then aisha (cast)", in12B)
	}
	if in12B[1].Role != domain.RoleCast || in12B[1].DisplayName != "aisha" {
		t.Errorf("participant ref = %+v", in12B[1])
	}
}

func TestCallSheetForUserIsPrivate(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	m, err := db.CreateProject(ctx, newUser(t, db, "madina"), "Private film")
	if err != nil {
		t.Fatal(err)
	}
	day := newDay(t, db, m.Project.ID, "2026-10-04", "07:30", "")
	stranger := newUser(t, db, "stranger")

	for name, id := range map[string]string{
		"someone else's day": day.ID,
		"made-up uuid":       "6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f",
		"not a uuid":         "nope",
	} {
		if _, err := db.CallSheetForUser(ctx, stranger.ID, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

func TestTodayForUser(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	// Aisha is cast in two films. Every user in this test is new, so the only
	// days she can see are the ones created below.
	aishaUser := newUser(t, db, "aisha")
	filmA, err := db.CreateProject(ctx, newUser(t, db, "madina"), "A film")
	if err != nil {
		t.Fatal(err)
	}
	filmB, err := db.CreateProject(ctx, newUser(t, db, "timur"), "B film")
	if err != nil {
		t.Fatal(err)
	}
	inA := joinAs(t, db, aishaUser, filmA.Project.ID, domain.RoleCast)
	inB := joinAs(t, db, aishaUser, filmB.Project.ID, domain.RoleCast)

	past := newDay(t, db, filmA.Project.ID, "2026-11-01", "08:00", "Yesterday's set")
	aMorning := newDay(t, db, filmA.Project.ID, "2026-11-02", "07:30", "Kok-Tobe")
	bAfternoon := newDay(t, db, filmB.Project.ID, "2026-11-02", "14:00", "Studio B")
	aLater := newDay(t, db, filmA.Project.ID, "2026-11-10", "09:00", "Tram stop")

	newScene(t, db, filmA.Project.ID, aMorning.ID, "5", inA.ID)
	newScene(t, db, filmA.Project.ID, aMorning.ID, "3", inA.ID)
	newScene(t, db, filmA.Project.ID, aMorning.ID, "4")       // not hers
	newScene(t, db, filmA.Project.ID, past.ID, "1", inA.ID)   // another date
	newScene(t, db, filmA.Project.ID, aLater.ID, "8", inA.ID) // another date
	putOnSetAllDay(t, db, bAfternoon.ID, inB.ID)

	// A film she is not in, shooting the same day, must not show up.
	foreign, err := db.CreateProject(ctx, newUser(t, db, "stranger"), "Not hers")
	if err != nil {
		t.Fatal(err)
	}
	newDay(t, db, foreign.Project.ID, "2026-11-02", "05:00", "Elsewhere")

	t.Run("days today, across projects, ordered by call time", func(t *testing.T) {
		date, days, err := db.TodayForUser(ctx, aishaUser.ID, "2026-11-02")
		if err != nil {
			t.Fatal(err)
		}
		if date != "2026-11-02" || len(days) != 2 {
			t.Fatalf("date %q, %d days; want 2026-11-02 with 2 days", date, len(days))
		}
		first, second := days[0], days[1]
		if first.Day.ID != aMorning.ID || second.Day.ID != bAfternoon.ID {
			t.Errorf("order = %s, %s; want the 07:30 day then the 14:00 day", first.Day.CallTime, second.Day.CallTime)
		}
		if first.ProjectName != "A film" || first.OnSetAllDay {
			t.Errorf("first = %+v", first)
		}
		if len(first.MyScenes) != 2 || first.MyScenes[0].Number != "3" || first.MyScenes[1].Number != "5" {
			t.Errorf("my scenes = %+v, want 3 then 5 (never 4)", first.MyScenes)
		}
		// In film B she has no scenes but is on set all day; the day still shows.
		if !second.OnSetAllDay || len(second.MyScenes) != 0 {
			t.Errorf("second = %+v, want on set all day with no scenes", second)
		}
	})

	t.Run("nothing today falls forward to the next date", func(t *testing.T) {
		date, days, err := db.TodayForUser(ctx, aishaUser.ID, "2026-11-03")
		if err != nil {
			t.Fatal(err)
		}
		if date != "2026-11-10" || len(days) != 1 || days[0].Day.ID != aLater.ID {
			t.Errorf("date %q, days %+v; want only the day on 2026-11-10", date, days)
		}
		if len(days) == 1 && (len(days[0].MyScenes) != 1 || days[0].MyScenes[0].Number != "8") {
			t.Errorf("my scenes = %+v, want scene 8", days[0].MyScenes)
		}
	})

	t.Run("nothing upcoming at all", func(t *testing.T) {
		date, days, err := db.TodayForUser(ctx, aishaUser.ID, "2027-01-01")
		if err != nil {
			t.Fatal(err)
		}
		if date != "" || len(days) != 0 {
			t.Errorf("date %q, %d days; want nothing", date, len(days))
		}
	})

	t.Run("someone in no project", func(t *testing.T) {
		date, days, err := db.TodayForUser(ctx, newUser(t, db, "nobody").ID, "2026-11-02")
		if err != nil || date != "" || len(days) != 0 {
			t.Errorf("got %q, %v, %v", date, days, err)
		}
	})
}
