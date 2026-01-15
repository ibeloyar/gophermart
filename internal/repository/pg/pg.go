package pg

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	migrationsTable = "schema_migrations"
	schemaName      = "public"
	migrationsPath  = "./migrations"

	maxAttempts = 3
)

type Repository struct {
	databaseURI string
	db          *sql.DB
}

func New(databaseURI string) (*Repository, error) {
	pool, err := pgxpool.New(context.Background(), databaseURI)
	if err != nil {
		return nil, err
	}

	db := stdlib.OpenDBFromPool(pool)

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: migrationsTable,
		SchemaName:      schemaName,
	})
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+absPath, "postgres", driver)
	if err != nil {
		return nil, err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, err
	}

	return &Repository{
		databaseURI: databaseURI,
		db:          db,
	}, nil
}

func (s *Repository) Ping() error {
	return s.db.Ping()
}

func (s *Repository) Shutdown() error {
	return s.db.Close()
}
