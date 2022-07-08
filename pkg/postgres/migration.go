package postgres

import (
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

func MigrationManager(dsn string, op func(*migrate.Migrate) error) error {
	wrapErr := func(err error, msg string) error {
		return fmt.Errorf("postgres.MigrationManager: %s, %w", msg, err)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return wrapErr(err, "invalid dsn format")
	}

	// go-migrate allows to specify a different migration table
	// than the default 'schema_migrations'. In this case, we want to use
	// a dedicated table to avoid potential clashing with the same tool running
	// on the same PostgreSQL database instance that is being used.
	q := u.Query()
	q.Add("x-migrations-table", "tiger_schema_migrations")
	u.RawQuery = q.Encode()

	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return wrapErr(err, "failed to create new iofs driver for reading migrations")
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, u.String())
	if err != nil {
		return wrapErr(err, "failed to create new migrate source for running db migrations")
	}
	if err := op(m); err != nil {
		return wrapErr(err, "failed to execute migration operation")
	}
	return nil
}

func MigrationUp(m *migrate.Migrate) error {
	wrapErr := func(err error, msg string) error {
		return fmt.Errorf("postgres.MigrationUp: %s, %w", msg, err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return wrapErr(err, "failed to execute up migrations")
	}

	return nil
}

func MigrationDown(m *migrate.Migrate) error {
	wrapErr := func(err error, msg string) error {
		return fmt.Errorf("postgres.MigrationDown: %s, %w", msg, err)
	}
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return wrapErr(err, "failed to execute down migrations")
	}
	return nil
}
