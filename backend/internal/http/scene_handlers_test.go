package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// --- fakes: what US-06 adds to the in-memory stores ---
//
// As with the other fakes, the SQL itself (ordering, locking, the change event
// row) is tested against Postgres in the store package. These are just enough
// state for the HTTP layer to be tested on top.

func (f *fakeProjects) ParticipantsForProject(_ context.Context, projectID string) ([]domain.Participant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Participant
	for _, pa := range f.participants {
		if pa.ProjectID == projectID {
			out = append(out, pa)
		}
	}
	return out, nil
}

func (f *fakeProjects) CreateParticipant(_ context.Context, projectID string, in domain.ParticipantInput) (domain.Participant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, pa := range f.participants {
		if in.Email != "" && pa.ProjectID == projectID && strings.EqualFold(pa.Email, in.Email) {
			return domain.Participant{}, store.ErrParticipantEmailTaken
		}
	}
	pa := domain.Participant{ID: f.id("dddddddd"), ProjectID: projectID, DisplayName: in.DisplayName, Email: in.Email, Role: in.Role}
	f.participants = append(f.participants, pa)
	return pa, nil
}

func (f *fakeProjects) refs(ids []string) []domain.ParticipantRef {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []domain.ParticipantRef{}
	for _, id := range ids {
		for _, pa := range f.participants {
			if pa.ID == id {
				out = append(out, domain.ParticipantRef{ID: pa.ID, DisplayName: pa.DisplayName, Role: pa.Role})
			}
		}
	}
	return out
}

func (f *fakeDays) DayForUser(ctx context.Context, userID, dayID string) (domain.ShootingDay, domain.Membership, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.days {
		if d.ID == dayID {
			m, err := f.projects.ProjectForUser(ctx, userID, d.ProjectID)
			if err != nil {
				return domain.ShootingDay{}, domain.Membership{}, store.ErrNotFound
			}
			return d, m, nil
		}
	}
	return domain.ShootingDay{}, domain.Membership{}, store.ErrNotFound
}

func (f *fakeDays) SetDayParticipants(_ context.Context, dayID string, ids []string) ([]domain.ParticipantRef, error) {
	refs := f.projects.refs(ids)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onSetAllDay[dayID] = refs
	return refs, nil
}

// fakeScenes is an in-memory SceneStore.
type fakeScenes struct {
	mu       sync.Mutex
	projects *fakeProjects
	days     *fakeDays
	scenes   []domain.Scene
	links    map[string][]string // scene id → participant ids
	events   []domain.Diff       // what UpdateScene logged
	n        int
}

func newFakeScenes(projects *fakeProjects, days *fakeDays) *fakeScenes {
	return &fakeScenes{projects: projects, days: days, links: map[string][]string{}}
}

func (f *fakeScenes) withPeople(s domain.Scene) domain.Scene {
	s.Participants = f.projects.refs(f.links[s.ID])
	return s
}

func (f *fakeScenes) ScenesForProject(_ context.Context, projectID string, filter store.SceneFilter) ([]domain.Scene, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Scene
	for _, s := range f.scenes {
		switch {
		case s.ProjectID != projectID:
		case filter.ShootingDayID != "" && s.ShootingDayID != filter.ShootingDayID:
		case filter.UnscheduledOnly && s.ShootingDayID != "":
		default:
			out = append(out, f.withPeople(s))
		}
	}
	return out, nil
}

func (f *fakeScenes) CreateScene(_ context.Context, projectID string, in domain.SceneInput) (domain.Scene, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.scenes {
		if s.ProjectID == projectID && s.Number == in.Number {
			return domain.Scene{}, store.ErrSceneNumberTaken
		}
	}
	f.n++
	s := domain.Scene{
		ID: fmt.Sprintf("ffffffff-0000-4000-8000-%012d", f.n), ProjectID: projectID,
		ShootingDayID: in.ShootingDayID, Number: in.Number, Description: in.Description,
	}
	f.scenes = append(f.scenes, s)
	return f.withPeople(s), nil
}

func (f *fakeScenes) SceneForUser(ctx context.Context, userID, sceneID string) (domain.Scene, domain.Membership, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.scenes {
		if s.ID == sceneID {
			m, err := f.projects.ProjectForUser(ctx, userID, s.ProjectID)
			if err != nil {
				return domain.Scene{}, domain.Membership{}, store.ErrNotFound
			}
			return f.withPeople(s), m, nil
		}
	}
	return domain.Scene{}, domain.Membership{}, store.ErrNotFound
}

func (f *fakeScenes) UpdateScene(_ context.Context, _ string, sceneID string, p domain.ScenePatch) (domain.Scene, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, s := range f.scenes {
		if s.ID != sceneID {
			continue
		}
		if diff := domain.DiffScene(s, p); len(diff) > 0 {
			f.events = append(f.events, diff)
		}
		if p.SetDay {
			f.scenes[i].ShootingDayID = p.ShootingDayID
		}
		if p.SetDescription {
			f.scenes[i].Description = p.Description
		}
		return f.withPeople(f.scenes[i]), nil
	}
	return domain.Scene{}, store.ErrNotFound
}

func (f *fakeScenes) SetSceneParticipants(_ context.Context, sceneID string, ids []string) (domain.Scene, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.scenes {
		if s.ID == sceneID {
			f.links[sceneID] = ids
			return f.withPeople(s), nil
		}
	}
	return domain.Scene{}, store.ErrNotFound
}

func (f *fakeScenes) ImpactInput(ctx context.Context, day domain.ShootingDay) (domain.ImpactInput, error) {
	f.mu.Lock()
	var in domain.ImpactInput
	for _, s := range f.scenes {
		if s.ShootingDayID == day.ID {
			in.Scenes = append(in.Scenes, s)
			for _, pid := range f.links[s.ID] {
				in.SceneLinks = append(in.SceneLinks, domain.SceneLink{SceneID: s.ID, ParticipantID: pid})
			}
		}
	}
	f.mu.Unlock()

	f.days.mu.Lock()
	for _, ref := range f.days.onSetAllDay[day.ID] {
		in.DayLinks = append(in.DayLinks, ref.ID)
	}
	f.days.mu.Unlock()

	in.Participants, _ = f.projects.ParticipantsForProject(ctx, day.ProjectID)
	return in, nil
}

// --- helpers ---

func (f crewFixture) participantsPath() string {
	return "/api/projects/" + f.project.ID + "/participants"
}
func (f crewFixture) scenesPath() string { return "/api/projects/" + f.project.ID + "/scenes" }

func (f crewFixture) addPerson(t *testing.T, name, role string) participantJSON {
	t.Helper()
	rec := do(t, f.h, http.MethodPost, f.participantsPath(), f.producer.Token,
		map[string]any{"display_name": name, "role": role, "email": nil})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add person: status %d, body %s", rec.Code, rec.Body)
	}
	return decode[participantJSON](t, rec)
}

func (f crewFixture) addScene(t *testing.T, number string, dayID *string) sceneJSON {
	t.Helper()
	rec := do(t, f.h, http.MethodPost, f.scenesPath(), f.producer.Token,
		map[string]any{"number": number, "description": "Scene " + number, "shooting_day_id": dayID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add scene: status %d, body %s", rec.Code, rec.Body)
	}
	return decode[sceneJSON](t, rec)
}

func (f crewFixture) participantID(t *testing.T, token string) string {
	t.Helper()
	return decode[projectJSON](t, do(t, f.h, http.MethodGet, "/api/projects/"+f.project.ID, token, nil)).MyParticipantID
}

func errorOf(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) apiError {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d; body %s", rec.Code, status, rec.Body)
	}
	e := decode[apiError](t, rec)
	if e.Error.Code != code {
		t.Fatalf("code %q, want %q; body %s", e.Error.Code, code, rec.Body)
	}
	return e
}

// --- US-06: people ---

func TestProducerAddsAPersonByName(t *testing.T) {
	f := newCrew(t)

	ruslan := f.addPerson(t, "  Ruslan ", "crew")
	if ruslan.DisplayName != "Ruslan" || ruslan.HasAccount || ruslan.Email != nil || ruslan.Role != "crew" {
		t.Errorf("ruslan = %+v, want crew, no account, email null", ruslan)
	}

	// Everyone in the project can see the list; the stranger cannot.
	list := decode[listResponse[participantJSON]](t, do(t, f.h, http.MethodGet, f.participantsPath(), f.crew.Token, nil))
	if len(list.Data) != 3 || list.NextCursor != nil {
		t.Errorf("list = %+v, want Madina, Daniyar, Ruslan on one page", list)
	}
	errorOf(t, do(t, f.h, http.MethodGet, f.participantsPath(), f.stranger.Token, nil), http.StatusNotFound, "project_not_found")
}

func TestAddPersonRules(t *testing.T) {
	f := newCrew(t)

	// Crew cannot add people.
	errorOf(t, do(t, f.h, http.MethodPost, f.participantsPath(), f.crew.Token,
		map[string]any{"display_name": "Ruslan", "role": "crew"}), http.StatusForbidden, "forbidden")

	e := errorOf(t, do(t, f.h, http.MethodPost, f.participantsPath(), f.producer.Token,
		map[string]any{"display_name": "", "role": "producer", "email": "nope"}), http.StatusUnprocessableEntity, "validation_failed")
	for _, field := range []string{"display_name", "role", "email"} {
		if e.Error.Fields[field] == "" {
			t.Errorf("no error for %s: %v", field, e.Error.Fields)
		}
	}

	// Daniyar is already in the project with this email.
	errorOf(t, do(t, f.h, http.MethodPost, f.participantsPath(), f.producer.Token,
		map[string]any{"display_name": "Dan", "role": "crew", "email": "DANIYAR@example.kz"}), http.StatusConflict, "participant_email_taken")
}

func TestSetDayCrewShowsOnTheCallSheet(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-11-02", "07:30", "Kok-Tobe")
	ruslan := f.addPerson(t, "Ruslan", "crew")
	path := "/api/days/" + day.ID + "/participants"

	rec := do(t, f.h, http.MethodPut, path, f.producer.Token, map[string]any{"participant_ids": []string{ruslan.ID, ruslan.ID}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	got := decode[struct {
		OnSetAllDay []participantRefJSON `json:"on_set_all_day"`
	}](t, rec)
	if len(got.OnSetAllDay) != 1 || got.OnSetAllDay[0].DisplayName != "Ruslan" {
		t.Errorf("on_set_all_day = %+v, want Ruslan once", got.OnSetAllDay)
	}
	sheet := decode[callSheetJSON](t, do(t, f.h, http.MethodGet, "/api/days/"+day.ID, f.crew.Token, nil))
	if len(sheet.OnSetAllDay) != 1 {
		t.Errorf("call sheet on_set_all_day = %+v", sheet.OnSetAllDay)
	}

	// Someone from outside the project is named in the error.
	e := errorOf(t, do(t, f.h, http.MethodPut, path, f.producer.Token,
		map[string]any{"participant_ids": []string{"someone-else"}}), http.StatusUnprocessableEntity, "validation_failed")
	if !strings.Contains(e.Error.Fields["participant_ids"], "someone-else") {
		t.Errorf("fields = %v, want the unknown id named", e.Error.Fields)
	}
	// Leaving the list out is not "clear it".
	errorOf(t, do(t, f.h, http.MethodPut, path, f.producer.Token, map[string]any{}), http.StatusUnprocessableEntity, "validation_failed")

	errorOf(t, do(t, f.h, http.MethodPut, path, f.crew.Token, map[string]any{"participant_ids": []string{}}), http.StatusForbidden, "forbidden")
	errorOf(t, do(t, f.h, http.MethodPut, path, f.stranger.Token, map[string]any{"participant_ids": []string{}}), http.StatusNotFound, "day_not_found")
}

// --- US-06: scenes ---

func TestCreateScenesOnADayAndUnscheduled(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-11-02", "07:30", "Kok-Tobe")

	onDay := f.addScene(t, "12A", &day.ID)
	if onDay.ShootingDayID == nil || *onDay.ShootingDayID != day.ID || onDay.Participants == nil {
		t.Errorf("scene = %+v", onDay)
	}
	loose := f.addScene(t, "14", nil)
	if loose.ShootingDayID != nil {
		t.Errorf("unscheduled scene has day %v", *loose.ShootingDayID)
	}

	all := decode[listResponse[sceneJSON]](t, do(t, f.h, http.MethodGet, f.scenesPath(), f.crew.Token, nil))
	unscheduled := decode[listResponse[sceneJSON]](t, do(t, f.h, http.MethodGet, f.scenesPath()+"?unscheduled=true", f.crew.Token, nil))
	ofDay := decode[listResponse[sceneJSON]](t, do(t, f.h, http.MethodGet, f.scenesPath()+"?shooting_day_id="+day.ID, f.crew.Token, nil))
	if len(all.Data) != 2 || len(unscheduled.Data) != 1 || unscheduled.Data[0].Number != "14" ||
		len(ofDay.Data) != 1 || ofDay.Data[0].Number != "12A" {
		t.Errorf("all %d, unscheduled %+v, of day %+v", len(all.Data), unscheduled.Data, ofDay.Data)
	}
	// The JSON says null, not "".
	if body := do(t, f.h, http.MethodGet, f.scenesPath()+"?unscheduled=true", f.crew.Token, nil).Body.String(); !strings.Contains(body, `"shooting_day_id":null`) {
		t.Errorf("body = %s", body)
	}
}

func TestCreateSceneRules(t *testing.T) {
	f := newCrew(t)
	f.addScene(t, "14", nil)

	errorOf(t, do(t, f.h, http.MethodPost, f.scenesPath(), f.producer.Token,
		map[string]any{"number": "14"}), http.StatusConflict, "scene_number_taken")
	errorOf(t, do(t, f.h, http.MethodPost, f.scenesPath(), f.crew.Token,
		map[string]any{"number": "15"}), http.StatusForbidden, "forbidden")
	errorOf(t, do(t, f.h, http.MethodPost, f.scenesPath(), f.producer.Token,
		map[string]any{"number": "  "}), http.StatusUnprocessableEntity, "validation_failed")

	// A day of another project — here, one Madina produces herself.
	other := createProject(t, f.h, f.producer.Token, "Short film")
	rec := do(t, f.h, http.MethodPost, "/api/projects/"+other.ID+"/days", f.producer.Token,
		map[string]string{"shoot_date": "2026-12-01", "call_time": "09:00"})
	otherDay := decode[dayJSON](t, rec)
	for _, dayID := range []string{otherDay.ID, "not-a-uuid", "eeeeeeee-0000-4000-8000-999999999999"} {
		e := errorOf(t, do(t, f.h, http.MethodPost, f.scenesPath(), f.producer.Token,
			map[string]any{"number": "16", "shooting_day_id": dayID}), http.StatusUnprocessableEntity, "validation_failed")
		if e.Error.Fields["shooting_day_id"] == "" {
			t.Errorf("day %s: fields = %v", dayID, e.Error.Fields)
		}
	}

	errorOf(t, do(t, f.h, http.MethodGet, f.scenesPath()+"?unscheduled=yes", f.crew.Token, nil), http.StatusUnprocessableEntity, "validation_failed")
	errorOf(t, do(t, f.h, http.MethodGet, f.scenesPath()+"?shooting_day_id=x", f.crew.Token, nil), http.StatusUnprocessableEntity, "validation_failed")
}

func TestPatchSceneMovesAndLogsOnlyRealChanges(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-11-02", "07:30", "Kok-Tobe")
	scene := f.addScene(t, "14", nil)
	path := "/api/scenes/" + scene.ID
	scenes := f.deps.Scenes.(*fakeScenes)

	rec := do(t, f.h, http.MethodPatch, path, f.producer.Token, map[string]any{"shooting_day_id": day.ID})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	if got := decode[sceneJSON](t, rec); got.ShootingDayID == nil || *got.ShootingDayID != day.ID {
		t.Errorf("scene = %+v, want on the day", got)
	}

	// null unschedules it again; an empty patch changes nothing.
	got := decode[sceneJSON](t, do(t, f.h, http.MethodPatch, path, f.producer.Token, `{"shooting_day_id": null}`))
	if got.ShootingDayID != nil {
		t.Errorf("after null: day = %v", *got.ShootingDayID)
	}
	if rec := do(t, f.h, http.MethodPatch, path, f.producer.Token, `{}`); rec.Code != http.StatusOK {
		t.Errorf("empty patch: status %d", rec.Code)
	}
	if len(scenes.events) != 2 {
		t.Errorf("%d change events, want 2 (on the day, then off it)", len(scenes.events))
	}

	e := errorOf(t, do(t, f.h, http.MethodPatch, path, f.producer.Token,
		map[string]any{"number": "15", "description": nil}), http.StatusUnprocessableEntity, "validation_failed")
	if e.Error.Fields["number"] == "" || e.Error.Fields["description"] == "" {
		t.Errorf("fields = %v, want number and description", e.Error.Fields)
	}
	errorOf(t, do(t, f.h, http.MethodPatch, path, f.crew.Token, map[string]any{"description": "x"}), http.StatusForbidden, "forbidden")
	errorOf(t, do(t, f.h, http.MethodPatch, path, f.stranger.Token, map[string]any{"description": "x"}), http.StatusNotFound, "scene_not_found")
}

func TestSetSceneParticipants(t *testing.T) {
	f := newCrew(t)
	scene := f.addScene(t, "12A", nil)
	aisha := f.addPerson(t, "Aisha", "cast")
	path := "/api/scenes/" + scene.ID + "/participants"

	rec := do(t, f.h, http.MethodPut, path, f.producer.Token, map[string]any{"participant_ids": []string{aisha.ID}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	if got := decode[sceneJSON](t, rec); len(got.Participants) != 1 || got.Participants[0].DisplayName != "Aisha" {
		t.Errorf("participants = %+v", got.Participants)
	}

	errorOf(t, do(t, f.h, http.MethodPut, path, f.producer.Token,
		map[string]any{"participant_ids": []string{"nobody"}}), http.StatusUnprocessableEntity, "validation_failed")
	errorOf(t, do(t, f.h, http.MethodPut, "/api/scenes/nope/participants", f.producer.Token,
		map[string]any{"participant_ids": []string{}}), http.StatusNotFound, "scene_not_found")
}

// --- US-07: the impact preview ---

type previewResponse struct {
	PreviewToken string `json:"preview_token"`
	ExpiresAt    string `json:"expires_at"`
	Change       struct {
		Kind          string                       `json:"kind"`
		ShootingDayID string                       `json:"shooting_day_id"`
		Diff          map[string]map[string]string `json:"diff"`
	} `json:"change"`
	Affected struct {
		Scenes       []sceneRefJSON            `json:"scenes"`
		Participants []affectedParticipantJSON `json:"participants"`
	} `json:"affected"`
	WillNotifyCount int `json:"will_notify_count"`
}

func preview(dayID string, to map[string]any) map[string]any {
	return map[string]any{"kind": "move_shooting_day", "shooting_day_id": dayID, "to": to}
}

func TestPreviewShowsWhoAMoveAffects(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-11-02", "07:30", "Kok-Tobe")
	daniyarID := f.participantID(t, f.crew.Token)
	madinaID := f.participantID(t, f.producer.Token)
	ruslan := f.addPerson(t, "Ruslan", "crew") // no account
	s12a := f.addScene(t, "12A", &day.ID)
	s13 := f.addScene(t, "13", &day.ID)
	f.addScene(t, "99", nil) // unscheduled: never affected

	for path, ids := range map[string][]string{
		"/api/scenes/" + s12a.ID + "/participants": {daniyarID, madinaID},
		"/api/scenes/" + s13.ID + "/participants":  {ruslan.ID, daniyarID},
		"/api/days/" + day.ID + "/participants":    {daniyarID},
	} {
		if rec := do(t, f.h, http.MethodPut, path, f.producer.Token, map[string]any{"participant_ids": ids}); rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d, body %s", path, rec.Code, rec.Body)
		}
	}

	rec := do(t, f.h, http.MethodPost, "/api/changes/preview", f.producer.Token,
		preview(day.ID, map[string]any{"shoot_date": "2026-11-04", "call_time": "07:30"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	got := decode[previewResponse](t, rec)

	if got.PreviewToken == "" || got.ExpiresAt == "" {
		t.Error("no token or expiry")
	}
	if _, err := f.deps.Previews.Verify(got.PreviewToken); err != nil {
		t.Errorf("token does not verify: %v", err)
	}
	// call_time is the same as now, so only the date is in the diff.
	if len(got.Change.Diff) != 1 || got.Change.Diff["shoot_date"]["from"] != "2026-11-02" || got.Change.Diff["shoot_date"]["to"] != "2026-11-04" {
		t.Errorf("diff = %v", got.Change.Diff)
	}
	if len(got.Affected.Scenes) != 2 || got.Affected.Scenes[0].Number != "12A" {
		t.Errorf("scenes = %+v", got.Affected.Scenes)
	}

	// Madina (producer, 12A), Daniyar (crew, all day + 12A + 13), Ruslan (crew, 13).
	people := got.Affected.Participants
	if len(people) != 3 || people[0].DisplayName != "Madina" || people[1].DisplayName != "Daniyar" || people[2].DisplayName != "Ruslan" {
		t.Fatalf("participants = %+v", people)
	}
	if d := people[1]; !d.Via.Day || strings.Join(d.Via.SceneNumbers, ",") != "12A,13" || !d.HasAccount {
		t.Errorf("daniyar = %+v, want once, with all three reasons", d)
	}
	if r := people[2]; r.HasAccount || r.Via.Day {
		t.Errorf("ruslan = %+v", r)
	}
	// Daniyar only: Madina is making the change, Ruslan has no account.
	if got.WillNotifyCount != 1 {
		t.Errorf("will_notify_count = %d, want 1", got.WillNotifyCount)
	}
}

func TestPreviewOfAnEmptyDayHasEmptyLists(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-11-02", "07:30", "Kok-Tobe")

	rec := do(t, f.h, http.MethodPost, "/api/changes/preview", f.producer.Token,
		preview(day.ID, map[string]any{"location": "Studio B"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	for _, want := range []string{`"scenes":[]`, `"participants":[]`, `"will_notify_count":0`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("body lacks %s: %s", want, rec.Body)
		}
	}
}

func TestPreviewRules(t *testing.T) {
	f := newCrew(t)
	day := f.addDay(t, "2026-11-02", "07:30", "Kok-Tobe")
	path := "/api/changes/preview"

	errorOf(t, do(t, f.h, http.MethodPost, path, f.producer.Token,
		preview(day.ID, map[string]any{"call_time": "07:30", "location": "Kok-Tobe"})), http.StatusUnprocessableEntity, "no_change")
	errorOf(t, do(t, f.h, http.MethodPost, path, f.producer.Token,
		preview(day.ID, map[string]any{})), http.StatusUnprocessableEntity, "validation_failed")
	e := errorOf(t, do(t, f.h, http.MethodPost, path, f.producer.Token,
		map[string]any{"kind": "delete_day", "shooting_day_id": day.ID, "to": map[string]any{"shoot_date": "4 Nov"}}),
		http.StatusUnprocessableEntity, "validation_failed")
	if e.Error.Fields["kind"] == "" || e.Error.Fields["to.shoot_date"] == "" {
		t.Errorf("fields = %v", e.Error.Fields)
	}
	errorOf(t, do(t, f.h, http.MethodPost, path, f.producer.Token, map[string]any{"kind": "move_shooting_day"}),
		http.StatusUnprocessableEntity, "validation_failed")

	errorOf(t, do(t, f.h, http.MethodPost, path, f.crew.Token,
		preview(day.ID, map[string]any{"shoot_date": "2026-11-04"})), http.StatusForbidden, "forbidden")
	errorOf(t, do(t, f.h, http.MethodPost, path, f.stranger.Token,
		preview(day.ID, map[string]any{"shoot_date": "2026-11-04"})), http.StatusNotFound, "day_not_found")
}
