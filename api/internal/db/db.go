// Package db manages the shared pgx connection pool used across handlers.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens a connection pool against DATABASE_URL and verifies it with
// a ping before returning, so startup fails fast on bad configuration.
//
// Uses DescribeExec instead of pgx's default (CacheStatement, which uses a
// named server-side prepared statement cached client-side per physical
// connection): Supabase's connection string can point at either its
// Session pooler (a stable connection per client — CacheStatement is fine)
// or its Transaction pooler (the underlying physical connection can be
// swapped between queries in the same session) — with the latter, a named
// prepared statement intermittently fails ("prepared statement does not
// exist") because it was prepared on a connection that got handed to
// someone else. DescribeExec still uses the extended (binary) protocol —
// each query is parsed/described/executed as an *unnamed* statement, so
// there's nothing tied to a specific physical connection to go stale.
//
// (An earlier version of this used QueryExecModeSimpleProtocol, which also
// avoids the stale-statement problem but falls back to text-only encoding —
// numeric columns like `price` come back as literal strings such as
// "120000.00" instead of being binary-decoded, which broke every Scan into
// an int64 money field as soon as a query actually returned rows. That
// went unnoticed because api/internal/testutil's pool doesn't go through
// NewPool. DescribeExec avoids that: binary decoding still applies.)
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
