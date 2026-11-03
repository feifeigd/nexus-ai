// Package testpg provides disposable PostgreSQL databases for integration
// tests. It isolates each test in its own database so concurrent packages never
// interfere through the shared chunk/approval tables.
package testpg

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/lib/pq"
)

// NewDatabase creates an isolated database derived from baseDSN and returns a
// DSN that targets it, plus a cleanup function that drops the database. The
// base DSN must reference a PostgreSQL instance whose role can create databases
// (the Docker Compose postgres user qualifies).
func NewDatabase(baseDSN string) (string, func(), error) {
	admin, err := sql.Open("postgres", baseDSN)
	if err != nil {
		return "", nil, fmt.Errorf("open admin connection: %w", err)
	}
	if err := admin.Ping(); err != nil {
		_ = admin.Close()
		return "", nil, fmt.Errorf("ping admin connection: %w", err)
	}
	name := fmt.Sprintf("nexus_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		_ = admin.Close()
		return "", nil, fmt.Errorf("create test database: %w", err)
	}
	_ = admin.Close()

	dsn, err := withDatabase(baseDSN, name)
	if err != nil {
		dropDatabase(baseDSN, name)
		return "", nil, err
	}
	cleanup := func() { dropDatabase(baseDSN, name) }
	return dsn, cleanup, nil
}

func withDatabase(baseDSN, name string) (string, error) {
	u, err := url.Parse(baseDSN)
	if err != nil {
		return "", fmt.Errorf("parse base DSN: %w", err)
	}
	u.Path = "/" + name
	return u.String(), nil
}

func dropDatabase(baseDSN, name string) {
	admin, err := sql.Open("postgres", baseDSN)
	if err != nil {
		return
	}
	defer admin.Close()
	_, _ = admin.Exec(`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`)
}
