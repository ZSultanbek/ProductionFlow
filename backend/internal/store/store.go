package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgx pool. Queries hang off this type, one file per aggregate.
type DB struct {
	Pool *pgxpool.Pool
}

// Open parses the connection string and creates the pool. It does not connect —
// pgx dials lazily — so callers that need a live connection at boot should Ping.
func Open(ctx context.Context, databaseURL string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Ping checks the database is reachable. /api/health calls this.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close waits for in-flight queries and releases the pool.
func (db *DB) Close() {
	db.Pool.Close()
}
