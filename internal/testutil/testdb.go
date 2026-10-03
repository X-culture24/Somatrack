// Package testutil gives integration tests a private, fully migrated
// database: a schema named it_<random> inside TEST_DATABASE_URL, with every
// migrations/*.up.sql applied in order. It's dropped when the test ends, so
// runs never see each other's rows or the developer's data.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB opens a fresh migrated schema, or skips the test when
// TEST_DATABASE_URL isn't set (so `go test ./...` stays DB-free).
func DB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — run via `make test-integration`")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b := make([]byte, 5)
	_, _ = rand.Read(b)
	schema := "it_" + hex.EncodeToString(b)

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Extensions are database-wide; install pgcrypto once, serialised, so
	// parallel tests don't race on 000_initial's CREATE EXTENSION.
	if _, err := admin.Exec(ctx, `SELECT pg_advisory_lock(72707); CREATE EXTENSION IF NOT EXISTS "pgcrypto"; SELECT pg_advisory_unlock(72707)`); err != nil {
		admin.Close()
		t.Fatalf("ensure pgcrypto: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if os.Getenv("KEEP_TEST_SCHEMA") != "" {
			t.Logf("kept schema %s", schema)
		} else if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Logf("drop schema %s: %v", schema, err)
		}
		admin.Close()
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	// public stays on the path for the pgcrypto extension.
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	for _, f := range migrationFiles(t) {
		sql, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migration %s: %v", filepath.Base(f), err)
		}
	}
	return pool
}

// migrationFiles lists migrations/*.up.sql in numeric order.
func migrationFiles(t testing.TB) []string {
	_, here, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(here), "..", "..", "migrations")
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found in %s", dir)
	}
	num := func(p string) int {
		n, _ := strconv.Atoi(strings.SplitN(filepath.Base(p), "_", 2)[0])
		return n
	}
	sort.Slice(files, func(i, j int) bool { return num(files[i]) < num(files[j]) })
	return files
}

// MustExec runs SQL that a test's setup depends on.
func MustExec(t testing.TB, pool *pgxpool.Pool, sql string, args ...interface{}) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", firstLine(sql), err)
	}
}

// Scalar reads a single value.
func Scalar[T any](t testing.TB, pool *pgxpool.Pool, sql string, args ...interface{}) T {
	t.Helper()
	var v T
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("query %q: %v", firstLine(sql), err)
	}
	return v
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i > 0 {
		s = s[:i]
	}
	return fmt.Sprintf("%.80s", s)
}
