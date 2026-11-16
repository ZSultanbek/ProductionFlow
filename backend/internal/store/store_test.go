package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// These tests talk to a real, migrated Postgres. They are skipped unless
// TEST_DATABASE_URL is set, so `go test ./...` stays fast and needs no setup:
//
//	make up migrate
//	TEST_DATABASE_URL=postgres://productionflow:productionflow@localhost:5432/productionflow?sslmode=disable \
//	    go test ./internal/store/
//
// Every test creates its own users with random emails, so they can run against
// a database that already has data in it, and against each other.
func testDB(t *testing.T) *DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func randomEmail(t *testing.T, name string) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return name + "+" + hex.EncodeToString(b) + "@test.kz"
}

func newUser(t *testing.T, db *DB, name string) domain.User {
	t.Helper()
	u, err := db.CreateUser(context.Background(), randomEmail(t, name), "not-a-real-hash", name)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestCreateUserEmailIsCaseInsensitivelyUnique(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	email := randomEmail(t, "aisha")

	if _, err := db.CreateUser(ctx, email, "h", "Aisha"); err != nil {
		t.Fatal(err)
	}
	upper := []byte(email)
	upper[0] -= 'a' - 'A'
	if _, err := db.CreateUser(ctx, string(upper), "h", "Aisha"); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("second account with %q: err = %v, want ErrEmailTaken", upper, err)
	}
	if _, _, err := db.UserByEmail(ctx, string(upper)); err != nil {
		t.Errorf("lookup by differently-cased email failed: %v", err)
	}
}

func TestUserByIDWithGarbageIDIsNotFound(t *testing.T) {
	db := testDB(t)
	if _, err := db.UserByID(context.Background(), "not-a-uuid"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestCreateProjectMakesCreatorProducer(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	madina := newUser(t, db, "madina")

	m, err := db.CreateProject(ctx, madina, "Dead Air")
	if err != nil {
		t.Fatal(err)
	}
	if m.Role != domain.RoleProducer || m.ParticipantID == "" {
		t.Errorf("membership = %+v, want producer with a participant id", m)
	}
	if len(m.Project.InviteCode) != 32 {
		t.Errorf("invite code = %q, want 32 hex characters", m.Project.InviteCode)
	}

	got, err := db.ProjectForUser(ctx, madina.ID, m.Project.ID)
	if err != nil || got.Role != domain.RoleProducer {
		t.Errorf("ProjectForUser = %+v, %v", got, err)
	}
}

func TestProjectForUserHidesProjectsYouAreNotIn(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	m, err := db.CreateProject(ctx, newUser(t, db, "madina"), "Private film")
	if err != nil {
		t.Fatal(err)
	}
	stranger := newUser(t, db, "stranger")

	for name, id := range map[string]string{
		"someone else's project": m.Project.ID,
		"made-up uuid":           "6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f",
		"not a uuid":             "nope",
	} {
		if _, err := db.ProjectForUser(ctx, stranger.ID, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

func TestProjectsForUserPagesNewestFirst(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	u := newUser(t, db, "pager")

	var made []string
	for _, name := range []string{"first", "second", "third"} {
		m, err := db.CreateProject(ctx, u, name)
		if err != nil {
			t.Fatal(err)
		}
		made = append(made, m.Project.Name)
	}

	var seen []string
	var after *Keyset
	for page := 0; page < 5; page++ {
		list, err := db.ProjectsForUser(ctx, u.ID, after, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range list {
			seen = append(seen, m.Project.Name)
		}
		if len(list) < 2 {
			break
		}
		last := list[len(list)-1]
		after = &Keyset{CreatedAt: last.Project.CreatedAt, ID: last.Project.ID}
	}

	want := []string{"third", "second", "first"}
	if len(seen) != len(want) {
		t.Fatalf("saw %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("position %d = %q, want %q (all: %v)", i, seen[i], want[i], seen)
		}
	}
}

func TestJoinProject(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	madina := newUser(t, db, "madina")
	m, err := db.CreateProject(ctx, madina, "Dead Air")
	if err != nil {
		t.Fatal(err)
	}
	projectID := m.Project.ID

	t.Run("new person is created with the chosen role", func(t *testing.T) {
		daniyar := newUser(t, db, "daniyar")
		p, created, err := db.JoinProject(ctx, daniyar, projectID, domain.RoleCrew)
		if err != nil || !created || p.Role != domain.RoleCrew || p.UserID != daniyar.ID {
			t.Fatalf("got %+v created=%v err=%v", p, created, err)
		}

		// Following the link again changes nothing, whatever role is asked for.
		again, created, err := db.JoinProject(ctx, daniyar, projectID, domain.RoleCast)
		if err != nil || created || again.ID != p.ID || again.Role != domain.RoleCrew {
			t.Errorf("second join: got %+v created=%v err=%v", again, created, err)
		}
	})

	t.Run("creator following their own link stays producer", func(t *testing.T) {
		p, created, err := db.JoinProject(ctx, madina, projectID, domain.RoleCrew)
		if err != nil || created || p.Role != domain.RoleProducer {
			t.Errorf("got %+v created=%v err=%v", p, created, err)
		}
	})

	t.Run("pre-added participant is linked, keeping the producer's role", func(t *testing.T) {
		aisha := newUser(t, db, "aisha")
		var preID string
		if err := db.Pool.QueryRow(ctx, `
			INSERT INTO participant (project_id, display_name, email, role)
			VALUES ($1, 'Aisha (added by producer)', upper($2), 'cast')
			RETURNING id::text`, projectID, aisha.Email).Scan(&preID); err != nil {
			t.Fatal(err)
		}

		p, created, err := db.JoinProject(ctx, aisha, projectID, domain.RoleCrew)
		if err != nil || created {
			t.Fatalf("got created=%v err=%v", created, err)
		}
		if p.ID != preID || p.UserID != aisha.ID || p.Role != domain.RoleCast {
			t.Errorf("got %+v; want row %s linked to %s, role cast", p, preID, aisha.ID)
		}
	})

	t.Run("unknown project", func(t *testing.T) {
		u := newUser(t, db, "lost")
		for _, id := range []string{"6f1b5f4e-1c2a-4d3b-9e5f-0a1b2c3d4e5f", "nope"} {
			if _, _, err := db.JoinProject(ctx, u, id, domain.RoleCrew); !errors.Is(err, ErrNotFound) {
				t.Errorf("project %q: err = %v, want ErrNotFound", id, err)
			}
		}
	})
}

// Two taps on the invite link, or two devices, at the same moment must still
// leave exactly one participant row.
func TestJoinProjectConcurrentlyCreatesOneRow(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	m, err := db.CreateProject(ctx, newUser(t, db, "madina"), "Race")
	if err != nil {
		t.Fatal(err)
	}
	joiner := newUser(t, db, "racer")

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := db.JoinProject(ctx, joiner, m.Project.ID, domain.RoleCrew)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent join failed: %v", err)
		}
	}

	var n int
	if err := db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM participant WHERE project_id = $1 AND user_id = $2`,
		m.Project.ID, joiner.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("%d participant rows, want 1", n)
	}
}

func TestProjectByInviteCode(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	m, err := db.CreateProject(ctx, newUser(t, db, "madina"), "Dead Air")
	if err != nil {
		t.Fatal(err)
	}

	p, err := db.ProjectByInviteCode(ctx, m.Project.InviteCode)
	if err != nil || p.ID != m.Project.ID {
		t.Errorf("got %+v, %v", p, err)
	}
	if _, err := db.ProjectByInviteCode(ctx, "00000000000000000000000000000000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown code: err = %v, want ErrNotFound", err)
	}
}

// Projects created in the same instant must still page without skipping or
// repeating one: the id breaks the tie in the keyset.
func TestProjectsForUserPagesThroughIdenticalTimestamps(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	u := newUser(t, db, "sametime")

	for i := 0; i < 5; i++ {
		if _, err := db.CreateProject(ctx, u, "same instant"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Pool.Exec(ctx, `
		UPDATE project SET created_at = '2026-11-02T09:00:00Z'
		WHERE id IN (SELECT project_id FROM participant WHERE user_id = $1)`, u.ID); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	var after *Keyset
	for page := 0; page < 10; page++ {
		list, err := db.ProjectsForUser(ctx, u.ID, after, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range list {
			if seen[m.Project.ID] {
				t.Fatalf("project %s returned twice", m.Project.ID)
			}
			seen[m.Project.ID] = true
		}
		if len(list) < 2 {
			break
		}
		last := list[len(list)-1]
		after = &Keyset{CreatedAt: last.Project.CreatedAt, ID: last.Project.ID}
	}
	if len(seen) != 5 {
		t.Errorf("saw %d projects across pages, want 5", len(seen))
	}
}
