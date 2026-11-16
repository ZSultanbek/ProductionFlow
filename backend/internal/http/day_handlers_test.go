package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// fakeDays is an in-memory DayStore. It asks fakeProjects who is a member, the
// way the real queries join on participant. The real SQL (ordering, scene
// counts, the fall-forward to the next date) is covered by the store tests
// against Postgres; these tests are about the HTTP layer on top.
type fakeDays struct {
	mu       sync.Mutex
	projects *fakeProjects
	days     []domain.ShootingDay
	n        int

	// What tests attach to a day, keyed by day id.
	onSetAllDay map[string][]domain.ParticipantRef
	scenes      map[string][]domain.CallSheetScene
}

func newFakeDays(projects *fakeProjects) *fakeDays {
	return &fakeDays{
		projects:    projects,
		onSetAllDay: map[string][]domain.ParticipantRef{},
		scenes:      map[string][]domain.CallSheetScene{},
	}
}

func dayLess(a, b domain.ShootingDay) bool {
	if a.ShootDate != b.ShootDate {
		return a.ShootDate < b.ShootDate
	}
	if a.CallTime != b.CallTime {
		return a.CallTime < b.CallTime
	}
	return a.ID < b.ID
}

func (f *fakeDays) CreateDay(_ context.Context, projectID string, in domain.DayInput) (domain.ShootingDay, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	d := domain.ShootingDay{
		ID:        fmt.Sprintf("eeeeeeee-0000-4000-8000-%012d", f.n),
		ProjectID: projectID,
		ShootDate: in.ShootDate,
		CallTime:  in.CallTime,
		Location:  in.Location,
		UpdatedAt: time.Date(2026, 11, 2, 9, 0, 0, 0, time.FixedZone("Almaty", 5*3600)),
	}
	f.days = append(f.days, d)
	return d, nil
}

func (f *fakeDays) DaysForProject(_ context.Context, projectID, from string, after *store.DayKeyset, limit int) ([]domain.ShootingDay, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.ShootingDay
	for _, d := range f.days {
		if d.ProjectID != projectID || (from != "" && d.ShootDate < from) {
			continue
		}
		if after != nil && !dayLess(domain.ShootingDay{ShootDate: after.ShootDate, CallTime: after.CallTime, ID: after.ID}, d) {
			continue
		}
		d.SceneCount = len(f.scenes[d.ID])
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return dayLess(out[i], out[j]) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeDays) CallSheetForUser(ctx context.Context, userID, dayID string) (domain.CallSheet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.days {
		if d.ID != dayID {
			continue
		}
		m, err := f.projects.ProjectForUser(ctx, userID, d.ProjectID)
		if err != nil {
			return domain.CallSheet{}, store.ErrNotFound
		}
		return domain.CallSheet{
			Day:         d,
			ProjectName: m.Project.Name,
			MyRole:      m.Role,
			OnSetAllDay: f.onSetAllDay[d.ID],
			Scenes:      f.scenes[d.ID],
		}, nil
	}
	return domain.CallSheet{}, store.ErrNotFound
}

func (f *fakeDays) TodayForUser(ctx context.Context, userID, date string) (string, []domain.AgendaDay, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Same rule as the real query: the earliest date >= date with any day in
	// one of the user's projects, then every such day on that date.
	target := ""
	var mine []domain.AgendaDay
	for _, d := range f.days {
		m, err := f.projects.ProjectForUser(ctx, userID, d.ProjectID)
		if err != nil || d.ShootDate < date {
			continue
		}
		if target == "" || d.ShootDate < target {
			target = d.ShootDate
		}
		mine = append(mine, domain.AgendaDay{Day: d, ProjectName: m.Project.Name})
	}
	var out []domain.AgendaDay
	for _, a := range mine {
		if a.Day.ShootDate == target {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return dayLess(out[i].Day, out[j].Day) })
	return target, out, nil
}

// --- helpers ---

type crewFixture struct {
	h        http.Handler
	deps     Deps
	producer authResponse
	crew     authResponse
	stranger authResponse
	project  projectJSON
}

// newCrew sets up the cast of most tests here: a project with its producer,
// one crew member who joined by invite, and someone who is not in it.
func newCrew(t *testing.T) crewFixture {
	t.Helper()
	deps := testDeps(nil)
	h := NewRouter(deps)
	f := crewFixture{
		h:        h,
		deps:     deps,
		producer: register(t, h, "madina@example.kz", "correct horse", "Madina"),
		crew:     register(t, h, "daniyar@example.kz", "correct horse", "Daniyar"),
		stranger: register(t, h, "stranger@example.kz", "correct horse", "Stranger"),
	}
	f.project = createProject(t, h, f.producer.Token, "Dead Air")
	join(t, h, f.crew.Token, *f.project.InviteCode, "crew", http.StatusCreated)
	return f
}

func (f crewFixture) daysPath() string { return "/api/projects/" + f.project.ID + "/days" }

func (f crewFixture) addDay(t *testing.T, date, callTime, location string) dayJSON {
	t.Helper()
	rec := do(t, f.h, http.MethodPost, f.daysPath(), f.producer.Token, map[string]string{
		"shoot_date": date, "call_time": callTime, "location": location,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add day: status %d, body %s", rec.Code, rec.Body)
	}
	return decode[dayJSON](t, rec)
}

// --- US-04: add a shooting day ---

func TestCreateDayAsProducer(t *testing.T) {
	f := newCrew(t)

	d := f.addDay(t, "2026-10-04", "07:30", "  Kok-Tobe, upper lot ")

	if d.ID == "" || d.ProjectID != f.project.ID {
		t.Errorf("day = %+v", d)
	}
	if d.ShootDate != "2026-10-04" || d.CallTime != "07:30" || d.Location != "Kok-Tobe, upper lot" {
		t.Errorf("day = %+v; date and call time must come back exactly as sent, location trimmed", d)
	}
	if d.SceneCount != 0 {
		t.Errorf("scene_count = %d, want 0 for a new day", d.SceneCount)
	}
	// Instants are UTC with a Z; the fake produced +05:00.
	if d.UpdatedAt != "2026-11-02T04:00:00Z" {
		t.Errorf("updated_at = %q, want UTC RFC 3339", d.UpdatedAt)
	}
}

// US-04's acceptance scenario: the day the producer saved is visible to every
// participant with the same call time.
func TestCreatedDayIsVisibleToCrewWithTheSameCallTime(t *testing.T) {
	f := newCrew(t)
	created := f.addDay(t, "2026-10-20", "07:00", "Studio 3")

	rec := do(t, f.h, http.MethodGet, f.daysPath(), f.crew.Token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	list := decode[listResponse[dayJSON]](t, rec)
	if len(list.Data) != 1 || list.Data[0] != created {
		t.Errorf("crew sees %+v, want exactly %+v", list.Data, created)
	}
}

func TestCreateDayIsProducerOnly(t *testing.T) {
	f := newCrew(t)
	body := map[string]string{"shoot_date": "2026-10-04", "call_time": "07:30", "location": "x"}

	rec := do(t, f.h, http.MethodPost, f.daysPath(), f.crew.Token, body)
	if rec.Code != http.StatusForbidden || decode[apiError](t, rec).Error.Code != "forbidden" {
		t.Errorf("crew: status %d body %s; want 403 forbidden", rec.Code, rec.Body)
	}

	// Someone outside the project must not even learn that it exists.
	rec = do(t, f.h, http.MethodPost, f.daysPath(), f.stranger.Token, body)
	if rec.Code != http.StatusNotFound || decode[apiError](t, rec).Error.Code != "project_not_found" {
		t.Errorf("stranger: status %d body %s; want 404 project_not_found", rec.Code, rec.Body)
	}

	if rec := do(t, f.h, http.MethodPost, f.daysPath(), "", body); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status %d, want 401", rec.Code)
	}

	// None of those attempts created anything.
	list := decode[listResponse[dayJSON]](t, do(t, f.h, http.MethodGet, f.daysPath(), f.producer.Token, nil))
	if len(list.Data) != 0 {
		t.Errorf("%d days exist after three refused attempts", len(list.Data))
	}
}

// A crew member sending garbage gets 403, not a list of what is wrong with a
// request they were never allowed to make.
func TestCreateDayChecksPermissionBeforeInput(t *testing.T) {
	f := newCrew(t)
	rec := do(t, f.h, http.MethodPost, f.daysPath(), f.crew.Token, map[string]string{"shoot_date": "nonsense"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("status %d, want 403", rec.Code)
	}
}

func TestCreateDayValidation(t *testing.T) {
	f := newCrew(t)

	rec := do(t, f.h, http.MethodPost, f.daysPath(), f.producer.Token, map[string]string{
		"shoot_date": "04.10.2026", "call_time": "7:30", "location": strings.Repeat("x", 201),
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422; body %s", rec.Code, rec.Body)
	}
	e := decode[apiError](t, rec)
	for _, field := range []string{"shoot_date", "call_time", "location"} {
		if e.Error.Fields[field] == "" {
			t.Errorf("fields.%s missing: %v", field, e.Error.Fields)
		}
	}

	rec = do(t, f.h, http.MethodPost, f.daysPath(), f.producer.Token, map[string]string{})
	e = decode[apiError](t, rec)
	if rec.Code != http.StatusUnprocessableEntity || e.Error.Fields["shoot_date"] == "" || e.Error.Fields["call_time"] == "" {
		t.Errorf("empty body: status %d, fields %v", rec.Code, e.Error.Fields)
	}
	if e.Error.Fields["location"] != "" {
		t.Errorf("location is optional but was reported: %v", e.Error.Fields)
	}
}

func TestTwoDaysOnTheSameDateAreAllowed(t *testing.T) {
	f := newCrew(t)
	a := f.addDay(t, "2026-10-04", "07:30", "Main unit")
	b := f.addDay(t, "2026-10-04", "07:30", "Second unit")
	if a.ID == b.ID {
		t.Error("second day on the same date was not created as its own day")
	}
}

// --- US-04: list days ---

func TestListDaysInShootingOrderWithFromAndPaging(t *testing.T) {
	f := newCrew(t)
	f.addDay(t, "2026-10-09", "13:00", "third")
	f.addDay(t, "2026-10-04", "15:00", "second")
	f.addDay(t, "2026-10-04", "07:30", "first")

	var got []string
	path := f.daysPath() + "?limit=2"
	for pages := 0; pages < 5; pages++ {
		rec := do(t, f.h, http.MethodGet, path, f.crew.Token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, body %s", rec.Code, rec.Body)
		}
		body := decode[listResponse[dayJSON]](t, rec)
		for _, d := range body.Data {
			got = append(got, d.Location)
		}
		if body.NextCursor == nil {
			break
		}
		path = f.daysPath() + "?limit=2&cursor=" + url.QueryEscape(*body.NextCursor)
	}
	if want := []string{"first", "second", "third"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("pages gave %v, want %v", got, want)
	}

	rec := do(t, f.h, http.MethodGet, f.daysPath()+"?from=2026-10-05", f.crew.Token, nil)
	list := decode[listResponse[dayJSON]](t, rec)
	if len(list.Data) != 1 || list.Data[0].Location != "third" {
		t.Errorf("from=2026-10-05 gave %+v, want only the day on the 9th", list.Data)
	}
}

func TestListDaysEmptyPrivateAndBadQuery(t *testing.T) {
	f := newCrew(t)

	if got := do(t, f.h, http.MethodGet, f.daysPath(), f.crew.Token, nil).Body.String(); got != `{"data":[],"next_cursor":null}` {
		t.Errorf("empty list body = %s", got)
	}

	rec := do(t, f.h, http.MethodGet, f.daysPath(), f.stranger.Token, nil)
	if rec.Code != http.StatusNotFound || decode[apiError](t, rec).Error.Code != "project_not_found" {
		t.Errorf("stranger: status %d body %s", rec.Code, rec.Body)
	}

	for query, field := range map[string]string{
		"from=yesterday":  "from",
		"from=2026-02-30": "from",
		"limit=0":         "limit",
		"cursor=garbage":  "cursor",
	} {
		rec := do(t, f.h, http.MethodGet, f.daysPath()+"?"+query, f.crew.Token, nil)
		if rec.Code != http.StatusUnprocessableEntity || decode[apiError](t, rec).Error.Fields[field] == "" {
			t.Errorf("?%s: status %d body %s; want 422 with fields.%s", query, rec.Code, rec.Body, field)
		}
	}
}

// A cursor is just base64 JSON, so anyone can make one up. Whatever is inside
// must be rejected as a bad cursor before it can reach SQL as a date, a time or
// a uuid — otherwise a tampered cursor is a 500.
func TestTamperedCursorsAreRejectedNotExecuted(t *testing.T) {
	f := newCrew(t)
	forge := func(json string) string { return base64.RawURLEncoding.EncodeToString([]byte(json)) }

	for name, tc := range map[string]struct{ path, cursor string }{
		"day cursor, bad date":       {f.daysPath(), forge(`{"d":"not-a-date","c":"07:30","id":"6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f"}`)},
		"day cursor, bad time":       {f.daysPath(), forge(`{"d":"2026-10-04","c":"25:99","id":"6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f"}`)},
		"day cursor, bad id":         {f.daysPath(), forge(`{"d":"2026-10-04","c":"07:30","id":"x'; DROP TABLE scene; --"}`)},
		"day cursor, wrong shape":    {f.daysPath(), forge(`{"t":"2026-11-02T09:00:00Z","id":"6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f"}`)},
		"project cursor, bad id":     {"/api/projects", forge(`{"t":"2026-11-02T09:00:00Z","id":"not-a-uuid"}`)},
		"project cursor, no time":    {"/api/projects", forge(`{"id":"6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f"}`)},
		"project cursor, not json":   {"/api/projects", forge(`hello`)},
		"project cursor, not base64": {"/api/projects", "%%%"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, f.h, http.MethodGet, tc.path+"?cursor="+url.QueryEscape(tc.cursor), f.crew.Token, nil)
			if rec.Code != http.StatusUnprocessableEntity || decode[apiError](t, rec).Error.Fields["cursor"] == "" {
				t.Errorf("status %d body %s; want 422 with fields.cursor", rec.Code, rec.Body)
			}
		})
	}
}

// --- US-05: the call sheet ---

func TestCallSheetShape(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-10-04", "07:30", "Kok-Tobe")

	fake := f.deps.Days.(*fakeDays)
	fake.onSetAllDay[day.ID] = []domain.ParticipantRef{{ID: "p-daniyar", DisplayName: "Daniyar", Role: domain.RoleCrew}}
	fake.scenes[day.ID] = []domain.CallSheetScene{
		{
			SceneRef:     domain.SceneRef{ID: "s-12a", Number: "12A", Description: "Rooftop argument, dusk"},
			Participants: []domain.ParticipantRef{{ID: "p-aisha", DisplayName: "Aisha", Role: domain.RoleCast}},
		},
		{SceneRef: domain.SceneRef{ID: "s-13", Number: "13", Description: "Insert: the note"}}, // nobody attached yet
	}

	rec := do(t, f.h, http.MethodGet, "/api/days/"+day.ID, f.crew.Token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	sheet := decode[callSheetJSON](t, rec)

	if sheet.ID != day.ID || sheet.ShootDate != "2026-10-04" || sheet.CallTime != "07:30" || sheet.Location != "Kok-Tobe" {
		t.Errorf("sheet = %+v", sheet)
	}
	if sheet.Project.ID != f.project.ID || sheet.Project.Name != "Dead Air" || sheet.Project.MyRole != "crew" {
		t.Errorf("project = %+v, want Dead Air seen as crew", sheet.Project)
	}
	if len(sheet.OnSetAllDay) != 1 || sheet.OnSetAllDay[0] != (participantRefJSON{ID: "p-daniyar", DisplayName: "Daniyar", Role: "crew"}) {
		t.Errorf("on_set_all_day = %+v", sheet.OnSetAllDay)
	}
	if len(sheet.Scenes) != 2 || sheet.Scenes[0].Number != "12A" || sheet.Scenes[0].Description != "Rooftop argument, dusk" {
		t.Fatalf("scenes = %+v", sheet.Scenes)
	}
	if len(sheet.Scenes[0].Participants) != 1 || sheet.Scenes[0].Participants[0].DisplayName != "Aisha" {
		t.Errorf("12A participants = %+v", sheet.Scenes[0].Participants)
	}

	// The producer opening the same sheet is told they are the producer.
	asProducer := decode[callSheetJSON](t, do(t, f.h, http.MethodGet, "/api/days/"+day.ID, f.producer.Token, nil))
	if asProducer.Project.MyRole != "producer" {
		t.Errorf("my_role for producer = %q", asProducer.Project.MyRole)
	}
}

// The client iterates these lists without null checks.
func TestCallSheetEmptyListsAreArraysNotNull(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-10-04", "07:30", "")
	f.deps.Days.(*fakeDays).scenes[day.ID] = []domain.CallSheetScene{
		{SceneRef: domain.SceneRef{ID: "s-1", Number: "1"}},
	}

	body := do(t, f.h, http.MethodGet, "/api/days/"+day.ID, f.crew.Token, nil).Body.String()
	for _, want := range []string{`"on_set_all_day":[]`, `"participants":[]`} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks %s: %s", want, body)
		}
	}
	if strings.Contains(body, "null") {
		t.Errorf("body contains a null: %s", body)
	}

	f.deps.Days.(*fakeDays).scenes[day.ID] = nil
	body = do(t, f.h, http.MethodGet, "/api/days/"+day.ID, f.crew.Token, nil).Body.String()
	if !strings.Contains(body, `"scenes":[]`) {
		t.Errorf("day with no scenes: body lacks \"scenes\":[]: %s", body)
	}
}

func TestCallSheetIsPrivate(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-10-04", "07:30", "")

	for name, tc := range map[string]struct{ id, token string }{
		"stranger":    {day.ID, f.stranger.Token},
		"unknown day": {"6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f", f.crew.Token},
		"not a uuid":  {"nope", f.crew.Token},
	} {
		rec := do(t, f.h, http.MethodGet, "/api/days/"+tc.id, tc.token, nil)
		if rec.Code != http.StatusNotFound || decode[apiError](t, rec).Error.Code != "day_not_found" {
			t.Errorf("%s: status %d body %s; want 404 day_not_found", name, rec.Code, rec.Body)
		}
	}
	if rec := do(t, f.h, http.MethodGet, "/api/days/"+day.ID, "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status %d, want 401", rec.Code)
	}
}

// --- US-05: today ---

func TestTodayShowsTodaysDays(t *testing.T) {
	f := newCrew(t)
	f.addDay(t, "2026-11-02", "14:00", "Studio B")
	morning := f.addDay(t, "2026-11-02", "07:30", "Kok-Tobe")
	f.addDay(t, "2026-11-10", "09:00", "Later")

	rec := do(t, f.h, http.MethodGet, "/api/me/today?date=2026-11-02", f.crew.Token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	today := decode[todayResponse](t, rec)

	if today.Date == nil || *today.Date != "2026-11-02" || !today.IsToday {
		t.Errorf("date = %v, is_today = %v; want 2026-11-02, true", today.Date, today.IsToday)
	}
	if len(today.Days) != 2 || today.Days[0].ID != morning.ID || today.Days[1].CallTime != "14:00" {
		t.Fatalf("days = %+v, want the 07:30 day then the 14:00 day", today.Days)
	}
	first := today.Days[0]
	if first.Project != (agendaProjectJSON{ID: f.project.ID, Name: "Dead Air"}) || first.Location != "Kok-Tobe" || first.ShootDate != "2026-11-02" {
		t.Errorf("first day = %+v", first)
	}
	if !strings.Contains(rec.Body.String(), `"my_scenes":[]`) {
		t.Errorf("my_scenes must be [] when empty: %s", rec.Body)
	}
}

// US-05 scenario 2: nothing today → the next shooting day, not an empty screen.
func TestTodayFallsForwardToTheNextShootingDate(t *testing.T) {
	f := newCrew(t)
	f.addDay(t, "2026-11-01", "08:00", "Yesterday")
	next := f.addDay(t, "2026-11-10", "09:00", "Tram stop")
	f.addDay(t, "2026-11-20", "09:00", "Even later")

	today := decode[todayResponse](t, do(t, f.h, http.MethodGet, "/api/me/today?date=2026-11-02", f.crew.Token, nil))

	if today.Date == nil || *today.Date != "2026-11-10" || today.IsToday {
		t.Errorf("date = %v, is_today = %v; want 2026-11-10, false", today.Date, today.IsToday)
	}
	if len(today.Days) != 1 || today.Days[0].ID != next.ID {
		t.Errorf("days = %+v, want only the day on the 10th", today.Days)
	}
}

func TestTodayWithNothingUpcoming(t *testing.T) {
	f := newCrew(t)
	f.addDay(t, "2026-11-01", "08:00", "Already shot")

	for name, token := range map[string]string{"member, all days past": f.crew.Token, "in no project with days": f.stranger.Token} {
		rec := do(t, f.h, http.MethodGet, "/api/me/today?date=2026-11-02", token, nil)
		if got := rec.Body.String(); rec.Code != http.StatusOK || got != `{"date":null,"is_today":false,"days":[]}` {
			t.Errorf("%s: status %d, body %s", name, rec.Code, got)
		}
	}
}

// The date is the phone's. The server refuses to guess it.
func TestTodayRequiresAValidDate(t *testing.T) {
	f := newCrew(t)
	for _, query := range []string{"", "?date=", "?date=today", "?date=2026-2-3", "?date=2026-02-30"} {
		rec := do(t, f.h, http.MethodGet, "/api/me/today"+query, f.crew.Token, nil)
		if rec.Code != http.StatusUnprocessableEntity || decode[apiError](t, rec).Error.Fields["date"] == "" {
			t.Errorf("%q: status %d body %s; want 422 with fields.date", query, rec.Code, rec.Body)
		}
	}
	if rec := do(t, f.h, http.MethodGet, "/api/me/today?date=2026-11-02", "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status %d, want 401", rec.Code)
	}
}
