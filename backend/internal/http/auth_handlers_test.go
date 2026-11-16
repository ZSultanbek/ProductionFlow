package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// fakeUsers is an in-memory UserStore with the same contract as store.DB:
// case-insensitive email uniqueness and the same sentinel errors.
type fakeUsers struct {
	mu     sync.Mutex
	byID   map[string]domain.User
	hashes map[string]string // user id -> password hash
	nextID int
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byID: map[string]domain.User{}, hashes: map[string]string{}}
}

func (f *fakeUsers) CreateUser(_ context.Context, email, hash, name string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if strings.EqualFold(u.Email, email) {
			return domain.User{}, store.ErrEmailTaken
		}
	}
	f.nextID++
	u := domain.User{
		ID:          fmt.Sprintf("00000000-0000-4000-8000-%012d", f.nextID),
		Email:       email,
		DisplayName: name,
		CreatedAt:   time.Date(2026, 11, 2, 9, 0, 0, 0, time.FixedZone("Almaty", 5*3600)),
	}
	f.byID[u.ID] = u
	f.hashes[u.ID] = hash
	return u, nil
}

func (f *fakeUsers) UserByEmail(_ context.Context, email string) (domain.User, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if strings.EqualFold(u.Email, email) {
			return u, f.hashes[u.ID], nil
		}
	}
	return domain.User{}, "", store.ErrNotFound
}

func (f *fakeUsers) UserByID(_ context.Context, id string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return domain.User{}, store.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) delete(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byID, id)
}

// --- helpers ---

type apiError struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	} `json:"error"`
}

func do(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	switch b := body.(type) {
	case nil:
	case string:
		buf.WriteString(b)
	default:
		if err := json.NewEncoder(&buf).Encode(b); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("unmarshal %q: %v", rec.Body.String(), err)
	}
	return v
}

func register(t *testing.T, h http.Handler, email, password, name string) authResponse {
	t.Helper()
	rec := do(t, h, http.MethodPost, "/api/auth/register", "", map[string]string{
		"email": email, "password": password, "display_name": name,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: status %d, body %s", rec.Code, rec.Body)
	}
	return decode[authResponse](t, rec)
}

// --- register ---

func TestRegisterCreatesAccountAndLogsIn(t *testing.T) {
	deps := testDeps(nil)
	h := NewRouter(deps)

	resp := register(t, h, "  Madina@Example.kz ", "correct horse", " Madina ")

	if resp.Token == "" {
		t.Fatal("no token")
	}
	if userID, err := deps.Tokens.Verify(resp.Token); err != nil || userID != resp.User.ID {
		t.Errorf("token does not verify to the new user: id=%q err=%v", userID, err)
	}
	if resp.User.Email != "Madina@Example.kz" || resp.User.DisplayName != "Madina" {
		t.Errorf("user not trimmed as expected: %+v", resp.User)
	}
	// Contract: instants are UTC with a Z, even though the fake returned +05:00.
	if resp.User.CreatedAt != "2026-11-02T04:00:00Z" {
		t.Errorf("created_at = %q, want UTC RFC 3339", resp.User.CreatedAt)
	}
	if _, err := time.Parse(time.RFC3339, resp.ExpiresAt); err != nil || !strings.HasSuffix(resp.ExpiresAt, "Z") {
		t.Errorf("expires_at = %q, want UTC RFC 3339", resp.ExpiresAt)
	}
}

func TestRegisterNeverReturnsThePassword(t *testing.T) {
	h := NewRouter(testDeps(nil))
	rec := do(t, h, http.MethodPost, "/api/auth/register", "", map[string]string{
		"email": "a@b.kz", "password": "correct horse", "display_name": "A",
	})
	if body := rec.Body.String(); strings.Contains(body, "correct horse") || strings.Contains(body, "$2a$") {
		t.Errorf("response leaks the password or its hash: %s", body)
	}
}

func TestRegisterValidation(t *testing.T) {
	h := NewRouter(testDeps(nil))
	rec := do(t, h, http.MethodPost, "/api/auth/register", "", map[string]string{
		"email": "nope", "password": "short", "display_name": "",
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	e := decode[apiError](t, rec)
	if e.Error.Code != "validation_failed" {
		t.Errorf("code = %q", e.Error.Code)
	}
	for _, f := range []string{"email", "password", "display_name"} {
		if e.Error.Fields[f] == "" {
			t.Errorf("fields.%s missing: %v", f, e.Error.Fields)
		}
	}
}

func TestRegisterRejectsMalformedJSON(t *testing.T) {
	h := NewRouter(testDeps(nil))
	rec := do(t, h, http.MethodPost, "/api/auth/register", "", `{"email": `)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if e := decode[apiError](t, rec); e.Error.Fields["body"] == "" {
		t.Errorf("fields.body missing: %+v", e.Error)
	}
}

func TestRegisterDuplicateEmailIgnoresCase(t *testing.T) {
	h := NewRouter(testDeps(nil))
	register(t, h, "aisha@mail.kz", "correct horse", "Aisha")

	rec := do(t, h, http.MethodPost, "/api/auth/register", "", map[string]string{
		"email": "AISHA@mail.kz", "password": "another one", "display_name": "Aisha 2",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if e := decode[apiError](t, rec); e.Error.Code != "email_taken" {
		t.Errorf("code = %q, want email_taken", e.Error.Code)
	}
}

// --- login ---

func TestLoginWithCorrectPassword(t *testing.T) {
	deps := testDeps(nil)
	h := NewRouter(deps)
	created := register(t, h, "madina@example.kz", "correct horse", "Madina")

	rec := do(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{
		"email": "MADINA@example.kz", "password": "correct horse",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	resp := decode[authResponse](t, rec)
	if resp.User.ID != created.User.ID {
		t.Errorf("logged in as %q, want %q", resp.User.ID, created.User.ID)
	}
	if _, err := deps.Tokens.Verify(resp.Token); err != nil {
		t.Errorf("token does not verify: %v", err)
	}
}

// Wrong password and unknown email must be indistinguishable to the client.
func TestLoginFailuresLookTheSame(t *testing.T) {
	h := NewRouter(testDeps(nil))
	register(t, h, "madina@example.kz", "correct horse", "Madina")

	wrongPassword := do(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{
		"email": "madina@example.kz", "password": "wrong horse",
	})
	unknownEmail := do(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{
		"email": "nobody@example.kz", "password": "correct horse",
	})

	for name, rec := range map[string]*httptest.ResponseRecorder{
		"wrong password": wrongPassword, "unknown email": unknownEmail,
	} {
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, rec.Code)
		}
	}
	if wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Errorf("bodies differ:\n  wrong password: %s\n  unknown email:  %s",
			wrongPassword.Body, unknownEmail.Body)
	}
	if e := decode[apiError](t, wrongPassword); e.Error.Code != "invalid_credentials" {
		t.Errorf("code = %q, want invalid_credentials", e.Error.Code)
	}
}

func TestLoginRequiresBothFields(t *testing.T) {
	h := NewRouter(testDeps(nil))
	rec := do(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{"email": "a@b.kz"})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if e := decode[apiError](t, rec); e.Error.Fields["password"] == "" {
		t.Errorf("fields.password missing: %+v", e.Error)
	}
}

// --- me and requireAuth ---

func TestMeReturnsTheTokensUser(t *testing.T) {
	h := NewRouter(testDeps(nil))
	created := register(t, h, "madina@example.kz", "correct horse", "Madina")

	rec := do(t, h, http.MethodGet, "/api/me", created.Token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if got := decode[meResponse](t, rec); got.User != created.User {
		t.Errorf("me = %+v, want %+v", got.User, created.User)
	}
}

func TestMeRejectsMissingOrBadTokens(t *testing.T) {
	h := NewRouter(testDeps(nil))

	for name, header := range map[string]string{
		"no header":     "",
		"wrong scheme":  "Basic dXNlcjpwYXNz",
		"empty bearer":  "Bearer ",
		"garbage token": "Bearer not.a.jwt",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if e := decode[apiError](t, rec); e.Error.Code != "unauthorized" {
				t.Errorf("code = %q, want unauthorized", e.Error.Code)
			}
			if rec.Header().Get("WWW-Authenticate") == "" {
				t.Error("no WWW-Authenticate header on 401")
			}
		})
	}
}

func TestBearerSchemeIsCaseInsensitive(t *testing.T) {
	h := NewRouter(testDeps(nil))
	created := register(t, h, "madina@example.kz", "correct horse", "Madina")

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "bearer "+created.Token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for lowercase scheme", rec.Code)
	}
}

func TestMeForDeletedAccountIsUnauthorized(t *testing.T) {
	deps := testDeps(nil)
	h := NewRouter(deps)
	created := register(t, h, "madina@example.kz", "correct horse", "Madina")

	deps.Users.(*fakeUsers).delete(created.User.ID)

	if rec := do(t, h, http.MethodGet, "/api/me", created.Token, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for a token whose account is gone", rec.Code)
	}
}
