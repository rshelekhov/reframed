// Package postgres implements postgres connection.
package postgres

import (
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Storage struct {
	DB *sqlx.DB
}

// NewPostgresStorage ...
func NewPostgresStorage(dsn string) (*Storage, error) {
	const op = "storage.postgres.NewStorage"

	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db}, nil
}

// Close closes the Postgres storage
func (s *Storage) Close() error {
	return s.DB.Close()
}