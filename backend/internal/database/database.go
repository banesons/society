package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNoDatabaseConfigured is returned when DATABASE_URL is not set.
	ErrNoDatabaseConfigured = errors.New("no database connection configured")
)

// DB encapsulates the PostgreSQL connection pool.
type DB struct {
	pool *pgxpool.Pool
}

// Connect initializes a connection pool to PostgreSQL using pgxpool.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return &DB{pool: nil}, nil
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	// PgBouncer pooler compatibility (prevents SQLSTATE 42P05 prepared statement collision)
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// Sane defaults for connection pooling
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection pool: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Ping checks database connectivity and reports round-trip latency.
func (d *DB) Ping(ctx context.Context) (int64, error) {
	if d == nil || d.pool == nil {
		return 0, ErrNoDatabaseConfigured
	}

	start := time.Now()
	if err := d.pool.Ping(ctx); err != nil {
		return 0, err
	}
	latency := time.Since(start).Milliseconds()

	return latency, nil
}

// Close gracefully closes the connection pool.
func (d *DB) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}

// IsConnected returns whether a connection pool exists.
func (d *DB) IsConnected() bool {
	return d != nil && d.pool != nil
}

// Pool returns the underlying pgxpool.Pool.
func (d *DB) Pool() *pgxpool.Pool {
	if d == nil {
		return nil
	}
	return d.pool
}
