package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lfordyce/tiger/internal/domain"
)

// psql -h localhost -U postgres -p 5432 -d homework
const defaultPostgresURL = "postgres://postgres:password@localhost:5432/homework?sslmode=disable"

type Repository struct {
	Conn *pgxpool.Pool
}

func NewRepository() (domain.Handler, func(), error) {
	conn, err := pgxpool.Connect(context.Background(), defaultPostgresURL)
	if err != nil {
		return Repository{}, func() {}, err
	}
	return Repository{Conn: conn}, func() {
		conn.Close()
	}, nil
}

func (r Repository) Process(req domain.Request) (float64, error) {
	var elapsed float64
	row := r.Conn.QueryRow(context.Background(), "SELECT * FROM bench($1::TEXT, $2::TIMESTAMPTZ, $3::TIMESTAMPTZ)",
		req.HostID, req.StartTime, req.EndTime)

	err := row.Scan(&elapsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return elapsed, fmt.Errorf("postgres: elapsed data not found")
	}
	if err != nil {
		return elapsed, fmt.Errorf("postgres: failed to query events table: %w", err)
	}
	return elapsed, nil
}
