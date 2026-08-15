// Package testutil spins up a real, throwaway Postgres database per test —
// the same manual verification workflow used throughout development
// (create DB, stub Supabase's auth schema, apply migrations, seed data,
// assert, drop DB), just automated and repeatable via `go test`.
package testutil

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewTestDB creates a uniquely-named database on a local Postgres server,
// stubs the `auth` schema Supabase provides in production (just enough
// for the `admins.auth_user_id` FK and the `auth.uid()` calls inside RLS
// policies to resolve), applies every migration in api/migrations in
// order, and returns a pool connected to it. The database is dropped when
// the test finishes.
//
// Skips the test (doesn't fail it) if no local Postgres is reachable, so
// `go test ./...` doesn't break on a machine/CI without one — set
// TEST_DATABASE_URL to point at a specific maintenance connection
// (a "postgres" database to CREATE DATABASE against) if the default
// (current OS user, localhost:5432) doesn't apply.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	maintURL := maintenanceURL()
	maintPool, err := pgxpool.New(ctx, maintURL)
	if err != nil {
		t.Skipf("no local postgres reachable at %s (%v) — skipping integration test", redactURL(maintURL), err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	pingErr := maintPool.Ping(pingCtx)
	cancel()
	if pingErr != nil {
		maintPool.Close()
		t.Skipf("no local postgres reachable at %s (%v) — skipping integration test", redactURL(maintURL), pingErr)
	}

	dbName := fmt.Sprintf("ecomwa_test_%d", time.Now().UnixNano())
	if _, err := maintPool.Exec(ctx, `CREATE DATABASE `+dbName); err != nil {
		maintPool.Close()
		t.Fatalf("failed to create test database %s: %v", dbName, err)
	}
	maintPool.Close()

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		cleanupPool, err := pgxpool.New(cleanupCtx, maintURL)
		if err != nil {
			return
		}
		defer cleanupPool.Close()
		// Terminate any lingering connections before dropping, or DROP
		// DATABASE fails with "database is being accessed by other users".
		_, _ = cleanupPool.Exec(cleanupCtx, `
			SELECT pg_terminate_backend(pid) FROM pg_stat_activity
			WHERE datname = $1 AND pid <> pg_backend_pid()
		`, dbName)
		_, _ = cleanupPool.Exec(cleanupCtx, `DROP DATABASE IF EXISTS `+dbName)
	})

	testURL := withDatabaseName(maintURL, dbName)
	pool, err := pgxpool.New(ctx, testURL)
	if err != nil {
		t.Fatalf("failed to connect to test database %s: %v", dbName, err)
	}
	t.Cleanup(pool.Close)

	applyAuthStub(ctx, t, pool)
	applyMigrations(ctx, t, pool)

	return pool
}

func maintenanceURL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	username := "postgres"
	if u, err := user.Current(); err == nil && u.Username != "" {
		username = u.Username
	}
	return fmt.Sprintf("postgres://%s@localhost:5432/postgres?sslmode=disable", username)
}

func withDatabaseName(rawURL, dbName string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Path = "/" + dbName
	return u.String()
}

func redactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "<unparseable>"
	}
	u.User = nil
	return u.String()
}

func applyAuthStub(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	statements := []string{
		`create schema auth`,
		`create table auth.users (id uuid primary key default gen_random_uuid())`,
		`create function auth.uid() returns uuid language sql stable as $$ select null::uuid $$`,
	}
	for _, stmt := range statements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("failed to apply auth schema stub (%q): %v", stmt, err)
		}
	}
}

func applyMigrations(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	entries, err := os.ReadDir(migrationsDir())
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // migration filenames are zero-padded (0001_, 0002_, ...)

	for _, name := range files {
		contents, err := os.ReadFile(filepath.Join(migrationsDir(), name))
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(contents)); err != nil {
			t.Fatalf("failed to apply migration %s: %v", name, err)
		}
	}
}

// migrationsDir resolves api/migrations relative to this source file's own
// location on disk, so it works regardless of which package's test binary
// (and therefore which working directory) calls NewTestDB.
func migrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
}
