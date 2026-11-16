package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// Keyset is a position in a list ordered newest first: everything strictly
// after (created_at, id) in that order. The http package turns it into the
// opaque cursor string clients see; nothing outside store and http knows it.
type Keyset struct {
	CreatedAt time.Time
	ID        string
}

// CreateProject creates a project and makes owner its producer, in one
// transaction — a project without a producer must never exist, even briefly.
func (db *DB) CreateProject(ctx context.Context, owner domain.User, name string) (domain.Membership, error) {
	var m domain.Membership

	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO project (name)
			VALUES ($1)
			RETURNING id::text, name, invite_code, created_at`,
			name).Scan(&m.Project.ID, &m.Project.Name, &m.Project.InviteCode, &m.Project.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert project: %w", err)
		}

		return tx.QueryRow(ctx, `
			INSERT INTO participant (project_id, user_id, display_name, email, role)
			VALUES ($1, $2, $3, $4, 'producer')
			RETURNING id::text`,
			m.Project.ID, owner.ID, owner.DisplayName, owner.Email).Scan(&m.ParticipantID)
	})
	if err != nil {
		return domain.Membership{}, fmt.Errorf("create project: %w", err)
	}

	m.Role = domain.RoleProducer
	return m, nil
}

// membershipColumns are selected by every query that returns a Membership, in
// the order scanMembership reads them. Callers alias project as p and
// participant as pa.
const membershipColumns = `p.id::text, p.name, p.invite_code, p.created_at, pa.id::text, pa.role`

func scanMembership(row pgx.Row) (domain.Membership, error) {
	var m domain.Membership
	var role string
	err := row.Scan(&m.Project.ID, &m.Project.Name, &m.Project.InviteCode, &m.Project.CreatedAt,
		&m.ParticipantID, &role)
	m.Role = domain.Role(role)
	return m, err
}

// ProjectsForUser lists the projects the user is a participant of, newest
// first. Pass after = nil for the first page. Returns up to limit rows; the
// caller asks for limit+1 to find out whether there is a next page.
func (db *DB) ProjectsForUser(ctx context.Context, userID string, after *Keyset, limit int) ([]domain.Membership, error) {
	var (
		afterTime *time.Time
		afterID   *string
	)
	if after != nil {
		afterTime, afterID = &after.CreatedAt, &after.ID
	}

	// The row comparison (a, b) < (x, y) is what makes keyset paging correct
	// when two projects share a created_at: the id breaks the tie.
	rows, err := db.Pool.Query(ctx, `
		SELECT `+membershipColumns+`
		FROM participant pa
		JOIN project p ON p.id = pa.project_id
		WHERE pa.user_id = $1
		  AND ($2::timestamptz IS NULL OR (p.created_at, p.id) < ($2, $3::uuid))
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $4`,
		userID, afterTime, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("projects for user: %w", err)
	}

	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Membership, error) {
		return scanMembership(row)
	})
	if err != nil {
		return nil, fmt.Errorf("projects for user: %w", err)
	}
	return list, nil
}

// ProjectForUser returns one project as seen by the user. ErrNotFound if the
// project does not exist, the id is not a UUID, or the user is not in it —
// deliberately the same answer for all three (docs/api.md, "Membership is private").
func (db *DB) ProjectForUser(ctx context.Context, userID, projectID string) (domain.Membership, error) {
	m, err := scanMembership(db.Pool.QueryRow(ctx, `
		SELECT `+membershipColumns+`
		FROM participant pa
		JOIN project p ON p.id = pa.project_id
		WHERE pa.user_id = $1 AND p.id = $2`,
		userID, projectID))

	if errors.Is(err, pgx.ErrNoRows) || isInvalidText(err) {
		return domain.Membership{}, ErrNotFound
	}
	if err != nil {
		return domain.Membership{}, fmt.Errorf("project for user: %w", err)
	}
	return m, nil
}

// ProjectByInviteCode finds the project an invite link points at.
func (db *DB) ProjectByInviteCode(ctx context.Context, code string) (domain.Project, error) {
	var p domain.Project
	err := db.Pool.QueryRow(ctx, `
		SELECT id::text, name, invite_code, created_at
		FROM project
		WHERE invite_code = $1`,
		code).Scan(&p.ID, &p.Name, &p.InviteCode, &p.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, ErrNotFound
	}
	if err != nil {
		return domain.Project{}, fmt.Errorf("project by invite code: %w", err)
	}
	return p, nil
}
