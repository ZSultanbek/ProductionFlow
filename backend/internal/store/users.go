package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/domain"
)

// IDs are selected as ::text so they scan into plain Go strings. pgx can scan
// uuid into other types too, but strings are what every layer above wants.
const userColumns = `id::text, email, display_name, created_at`

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	return u, err
}

// CreateUser inserts an account. Returns ErrEmailTaken if the email is already
// registered in any letter case.
func (db *DB) CreateUser(ctx context.Context, email, passwordHash, displayName string) (domain.User, error) {
	row := db.Pool.QueryRow(ctx, `
		INSERT INTO app_user (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING `+userColumns,
		email, passwordHash, displayName)

	u, err := scanUser(row)
	if isUniqueViolation(err, "app_user_email_idx") {
		return domain.User{}, ErrEmailTaken
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// UserByEmail returns the account and its password hash, matching the email
// case-insensitively. The hash is returned separately so it never travels
// around inside domain.User. ErrNotFound if there is no such account.
func (db *DB) UserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var (
		u    domain.User
		hash string
	)
	// lower(email) = lower($1) is exactly the expression app_user_email_idx is
	// built on, so this is an index lookup, not a scan.
	err := db.Pool.QueryRow(ctx, `
		SELECT `+userColumns+`, password_hash
		FROM app_user
		WHERE lower(email) = lower($1)`,
		email).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &hash)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", ErrNotFound
	}
	if err != nil {
		return domain.User{}, "", fmt.Errorf("user by email: %w", err)
	}
	return u, hash, nil
}

// UserByID returns the account with this id. ErrNotFound if there is none —
// including when id is not a valid UUID at all.
func (db *DB) UserByID(ctx context.Context, id string) (domain.User, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM app_user
		WHERE id = $1`,
		id)

	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidText(err) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("user by id: %w", err)
	}
	return u, nil
}
