// Package testdb prepares the PostgreSQL database used by integration tests.
package testdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// migrationLock is an arbitrary application-wide advisory-lock key.
const migrationLock = 7_061_001

// Migrate applies every migration to the database at url. Packages' tests run in parallel
// against the same database, so migrating is serialised with a PostgreSQL advisory lock:
// whichever package gets there first migrates, the others wait and find nothing to do.
func Migrate(t testing.TB, url string) {
	t.Helper()
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	lock, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, err := lock.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, migrationLock); err != nil {
		t.Fatal(err)
	}
	defer lock.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, migrationLock) //nolint:errcheck

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(db, migrationsDir()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

// migrationsDir locates ./migrations relative to this file, so it works from any package.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}
