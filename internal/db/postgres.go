// Package db provides Postgres connectivity and migration helpers.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julianstephens/formation/internal/config"
)

// Open creates and validates a pgxpool connection pool to Postgres.
// The caller must call pool.Close() during graceful shutdown.
func Open(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	// Conservative pool defaults; tune via config in later phases if needed.
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open pgxpool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
