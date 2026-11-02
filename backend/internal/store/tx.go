package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// WithTx runs fn inside one transaction: commit if fn returns nil, roll back
// otherwise (and if fn panics). Use it whenever a write touches two tables that
// must change together, like a project and its creator's participant row.
func (db *DB) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	// Rollback after a successful Commit is a no-op, so this is safe to defer
	// unconditionally — it only does something on the error and panic paths.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
