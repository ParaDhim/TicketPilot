package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context) (*DB, error) {
	url := os.Getenv("DB_URL")
	if url == "" {
		url = "postgres://user:pass@localhost:5432/ticketpilot?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	return &DB{Pool: pool}, nil
}
