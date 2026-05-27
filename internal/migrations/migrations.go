package migrations

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrationsFS embed.FS

func RunMigrations(DBDSN string) error {
	db, err := sql.Open("pgx", DBDSN)
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return err
	}
	defer db.Close()

	sourceDvr, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return err
	}
	defer sourceDvr.Close()

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDvr, "postgres", dbDriver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err == nil {
		return nil
	}

	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	} else {
		return err
	}
}
