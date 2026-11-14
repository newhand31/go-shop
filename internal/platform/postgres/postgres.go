// Package postgres opens the connection pool and applies migrations.
// It is infrastructure shared by main and the integration tests.
package postgres

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/newhand31/go-shop/db"
)

// Open creates a pgx pool with pgx's default size and pings it, so a wrong DATABASE_URL fails at startup instead of on
// the first request.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	return OpenWithMaxConns(ctx, url, 0)
}

// OpenWithMaxConns is Open with an explicit ceiling on connections (0 keeps pgx's default). The ceiling is a promise
// to the database: PostgreSQL allows 100 connections by default, and every API instance spends up to this many of them.
// A request that wants a connection when all are busy waits for one; it does not open a new one.
func OpenWithMaxConns(ctx context.Context, url string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate applies every pending goose migration embedded in the binary.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// goose speaks database/sql; OpenDBFromPool borrows connections from the same pgx pool.
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	// goose expects the .sql files at the root of the FS, so strip the migrations/ prefix.
	dir, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, dir)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
