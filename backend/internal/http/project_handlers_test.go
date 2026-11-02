package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// fakeProjects is an in-memory ProjectStore. It uses domain.DecideJoin for
// joining, exactly as store.DB does, so the handler tests and the real store
// cannot disagree about what joining means.
type fakeProjects struct {
	mu           sync.Mutex
	projects     map[string]domain.Project
	participants []domain.Participant
	n            int
	clock        time.Time
}

func newFakeProjects() *fakeProjects {
	return &fakeProjects{
		projects: map[string]domain.Project{},
		clock:    time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC),
	}
}

func (f *fakeProjects) id(prefix string) string {
	f.n++
	return fmt.Sprintf("%s-0000-4000-8000-%012d", prefix, f.n)
}

func (f *fakeProjects) CreateProject(_ context.Context, owner domain.User, name string) (domain.Membership, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clock = f.clock.Add(time.Minute)
	p := domain.Project{ID: f.id("aaaaaaaa"), Name: name, InviteCode: fmt.Sprintf("code%028d", f.n), CreatedAt: f.clock}
	f.projects[p.ID] = p
	pa := domain.Participant{ID: f.id("bbbbbbbb"), ProjectID: p.ID, UserID: owner.ID,
		DisplayName: owner.DisplayName, Email: owner.Email, Role: domain.RoleProducer}
	f.participants = append(f.participants, pa)
	return domain.Membership{Project: p, ParticipantID: pa.ID, Role: domain.RoleProducer}, nil
}

func (f *fakeProjects) ProjectsForUser(_ context.Context, userID string, after *store.Keyset, limit int) ([]domain.Membership, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Membership
	for _, pa := range f.participants {
		if pa.UserID != userID {
			continue
		}
		p := f.projects[pa.ProjectID]
		if after != nil && !(p.CreatedAt.Before(after.CreatedAt) ||
			(p.CreatedAt.Equal(after.CreatedAt) && p.ID < after.ID)) {
			continue
		}
		out = append(out, domain.Membership{Project: p, ParticipantID: pa.ID, Role: pa.Role})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Project.CreatedAt.After(out[j].Project.CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeProjects) ProjectForUser(_ context.Context, userID, projectID string) (domain.Membership, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, pa := range f.participants {
		if pa.UserID == userID && pa.ProjectID == projectID {
			return domain.Membership{Project: f.projects[projectID], ParticipantID: pa.ID, Role: pa.Role}, nil
		}
	}
	return domain.Membership{}, store.ErrNotFound
}

func (f *fakeProjects) ProjectByInviteCode(_ context.Context, code string) (domain.Project, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.projects {
		if p.InviteCode == code {
			return p, nil
		}
	}
	return domain.Project{}, store.ErrNotFound
}

func (f *fakeProjects) JoinProject(_ context.Context, user domain.User, projectID string, role domain.Role) (domain.Participant, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var people []domain.Participant
	for _, pa := range f.participants {
		if pa.ProjectID == projectID {
			people = append(people, pa)
		}
	}
	outcome, id := domain.DecideJoin(people, user.ID, user.Email)
	switch outcome {
	case domain.JoinAlreadyMember:
		for _, pa := range people {
			if pa.ID == id {
				return pa, false, nil
			}
		}
	case domain.JoinLinkExisting:
		for i := range f.participants {
			if f.participants[i].ID == id {
				f.participants[i].UserID = user.ID
				return f.participants[i], false, nil
			}
		}
	case domain.JoinCreate:
		pa := domain.Participant{ID: f.id("cccccccc"), ProjectID: projectID, UserID: user.ID,
			DisplayName: user.DisplayName, Email: user.Email, Role: role}
		f.participants = append(f.participants, pa)
		return pa, true, nil
	}
	return domain.Participant{}, false, store.ErrParticipantEmailTaken
}

// addByName is what the producer does in US-06: a participant with no account.
func (f *fakeProjects) addByName(projectID, name, email string, role domain.Role) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	pa := domain.Participant{ID: f.id("dddddddd"), ProjectID: projectID, DisplayName: name, Email: email, Role: role}
	f.participants = append(f.participants, pa)
	return pa.ID
}

// --- helpers ---

func createProject(t *testing.T, h http.Handler, token, name string) projectJSON {
	t.Helper()
	rec := do(t, h, http.MethodPost, "/api/projects", token, map[string]string{"name": name})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project: status %d, body %s", rec.Code, rec.Body)
	}
	return decode[projectJSON](t, rec)
}

func join(t *testing.T, h http.Handler, token, code, role string, wantStatus int) participantJSON {
	t.Helper()
	rec := do(t, h, http.MethodPost, "/api/invites/"+code+"/join", token, map[string]string{"role": role})
	if rec.Code != wantStatus {
		t.Fatalf("join: status %d, want %d, body %s", rec.Code, wantStatus, rec.Body)
	}
	return decode[participantJSON](t, rec)
}

// --- US-02: projects ---

func TestCreateProjectMakesCallerProducerWithInviteLink(t *testing.T) {
	h := NewRouter(testDeps(nil))
	madina := register(t, h, "madina@example.kz", "correct horse", "Madina")

	p := createProject(t, h, madina.Token, "  Dead Air  ")

	if p.Name != "Dead Air" || p.MyRole != "producer" || p.MyParticipantID == "" {
		t.Errorf("project = %+v", p)
	}
	if p.InviteCode == nil || p.InviteURL == nil {
		t.Fatalf("producer must see the invite: %+v", p)
	}
	if want := "https://web.test/join/" + *p.InviteCode; *p.InviteURL != want {
		t.Errorf("invite_url = %q, want %q", *p.InviteURL, want)
	}
}

func TestCreateProjectValidationAndAuth(t *testing.T) {
	h := NewRouter(testDeps(nil))
	madina := register(t, h, "madina@example.kz", "correct horse", "Madina")

	rec := do(t, h, http.MethodPost, "/api/projects", madina.Token, map[string]string{"name": "  "})
	if rec.Code != http.StatusUnprocessableEntity || decode[apiError](t, rec).Error.Fields["name"] == "" {
		t.Errorf("blank name: status %d, body %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodPost, "/api/projects", "", map[string]string{"name": "x"}); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status %d, want 401", rec.Code)
	}
}

func TestListProjectsShowsOnlyMineNewestFirstAndPages(t *testing.T) {
	h := NewRouter(testDeps(nil))
	madina := register(t, h, "madina@example.kz", "correct horse", "Madina")
	other := register(t, h, "other@example.kz", "correct horse", "Other")

	createProject(t, h, madina.Token, "first")
	createProject(t, h, other.Token, "not mine")
	createProject(t, h, madina.Token, "second")
	createProject(t, h, madina.Token, "third")

	var names []string
	path := "/api/projects?limit=2"
	for pages := 0; pages < 5; pages++ {
		rec := do(t, h, http.MethodGet, path, madina.Token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, body %s", rec.Code, rec.Body)
		}
		body := decode[listResponse[projectJSON]](t, rec)
		for _, p := range body.Data {
			names = append(names, p.Name)
		}
		if body.NextCursor == nil {
			break
		}
		path = "/api/projects?limit=2&cursor=" + url.QueryEscape(*body.NextCursor)
	}

	want := []string{"third", "second", "first"}
	if fmt.Sprint(names) != fmt.Sprint(want) {
		t.Errorf("pages gave %v, want %v", names, want)
	}
}

func TestListProjectsEmptyIsAnEmptyArrayNotNull(t *testing.T) {
	h := NewRouter(testDeps(nil))
	u := register(t, h, "new@example.kz", "correct horse", "New")

	rec := do(t, h, http.MethodGet, "/api/projects", u.Token, nil)
	if got := rec.Body.String(); got != `{"data":[],"next_cursor":null}` {
		t.Errorf("body = %s", got)
	}
}

func TestListProjectsRejectsBadPaging(t *testing.T) {
	h := NewRouter(testDeps(nil))
	u := register(t, h, "u@example.kz", "correct horse", "U")

	for _, q := range []string{"limit=0", "limit=101", "limit=ten", "cursor=garbage"} {
		rec := do(t, h, http.MethodGet, "/api/projects?"+q, u.Token, nil)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("?%s: status %d, want 422", q, rec.Code)
		}
	}
}

func TestGetProjectIsPrivate(t *testing.T) {
	h := NewRouter(testDeps(nil))
	madina := register(t, h, "madina@example.kz", "correct horse", "Madina")
	stranger := register(t, h, "stranger@example.kz", "correct horse", "Stranger")
	p := createProject(t, h, madina.Token, "Dead Air")

	if rec := do(t, h, http.MethodGet, "/api/projects/"+p.ID, madina.Token, nil); rec.Code != http.StatusOK {
		t.Errorf("member: status %d", rec.Code)
	}
	rec := do(t, h, http.MethodGet, "/api/projects/"+p.ID, stranger.Token, nil)
	if rec.Code != http.StatusNotFound || decode[apiError](t, rec).Error.Code != "project_not_found" {
		t.Errorf("non-member: status %d body %s; want 404 project_not_found", rec.Code, rec.Body)
	}
}

// --- US-03: invites ---

func TestInviteLookupIsPublic(t *testing.T) {
	h := NewRouter(testDeps(nil))
	madina := register(t, h, "madina@example.kz", "correct horse", "Madina")
	p := createProject(t, h, madina.Token, "Dead Air")

	rec := do(t, h, http.MethodGet, "/api/invites/"+*p.InviteCode, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	if got := decode[inviteJSON](t, rec); got.ProjectID != p.ID || got.ProjectName != "Dead Air" {
		t.Errorf("invite = %+v", got)
	}

	rec = do(t, h, http.MethodGet, "/api/invites/nope", "", nil)
	if rec.Code != http.StatusNotFound || decode[apiError](t, rec).Error.Code != "invite_not_found" {
		t.Errorf("unknown code: status %d body %s", rec.Code, rec.Body)
	}
}

func TestJoinCreatesThenIsIdempotent(t *testing.T) {
	h := NewRouter(testDeps(nil))
	madina := register(t, h, "madina@example.kz", "correct horse", "Madina")
	daniyar := register(t, h, "daniyar@example.kz", "correct horse", "Daniyar")
	p := createProject(t, h, madina.Token, "Dead Air")

	first := join(t, h, daniyar.Token, *p.InviteCode, "crew", http.StatusCreated)
	if first.Role != "crew" || !first.HasAccount || first.DisplayName != "Daniyar" || first.ProjectID != p.ID {
		t.Errorf("participant = %+v", first)
	}

	// Following the link again, even asking for another role, changes nothing.
	again := join(t, h, daniyar.Token, *p.InviteCode, "cast", http.StatusOK)
	if again.ID != first.ID || again.Role != "crew" {
		t.Errorf("second join = %+v, want unchanged %+v", again, first)
	}

	// The project now shows up for them, as crew, without the invite code.
	rec := do(t, h, http.MethodGet, "/api/projects/"+p.ID, daniyar.Token, nil)
	got := decode[projectJSON](t, rec)
	if got.MyRole != "crew" || got.InviteCode != nil || got.InviteURL != nil {
		t.Errorf("crew view of project = %+v; must not include the invite", got)
	}
}

func TestJoinLinksParticipantTheProducerAddedByEmail(t *testing.T) {
	deps := testDeps(nil)
	h := NewRouter(deps)
	madina := register(t, h, "madina@example.kz", "correct horse", "Madina")
	p := createProject(t, h, madina.Token, "Dead Air")
	preID := deps.Projects.(*fakeProjects).addByName(p.ID, "Aisha", "AISHA@example.kz", domain.RoleCast)

	aisha := register(t, h, "aisha@example.kz", "correct horse", "Aisha B.")
	got := join(t, h, aisha.Token, *p.InviteCode, "crew", http.StatusOK)

	if got.ID != preID || got.Role != "cast" || !got.HasAccount {
		t.Errorf("got %+v; want the pre-added row %s, still cast, now with an account", got, preID)
	}
}

func TestJoinRejectsProducerRoleAndBadInput(t *testing.T) {
	h := NewRouter(testDeps(nil))
	madina := register(t, h, "madina@example.kz", "correct horse", "Madina")
	u := register(t, h, "u@example.kz", "correct horse", "U")
	p := createProject(t, h, madina.Token, "Dead Air")

	for _, role := range []string{"producer", "", "director"} {
		rec := do(t, h, http.MethodPost, "/api/invites/"+*p.InviteCode+"/join", u.Token, map[string]string{"role": role})
		if rec.Code != http.StatusUnprocessableEntity || decode[apiError](t, rec).Error.Fields["role"] == "" {
			t.Errorf("role %q: status %d body %s; want 422 with fields.role", role, rec.Code, rec.Body)
		}
	}

	if rec := do(t, h, http.MethodPost, "/api/invites/"+*p.InviteCode+"/join", "", map[string]string{"role": "crew"}); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status %d, want 401", rec.Code)
	}

	rec := do(t, h, http.MethodPost, "/api/invites/nope/join", u.Token, map[string]string{"role": "crew"})
	if rec.Code != http.StatusNotFound || decode[apiError](t, rec).Error.Code != "invite_not_found" {
		t.Errorf("unknown code: status %d body %s", rec.Code, rec.Body)
	}
}
