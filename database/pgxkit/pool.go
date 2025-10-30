package pgxkit

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {
	// One of:
	DatabaseURL string // e.g. postgres://user:pass@host:5432/db?sslmode=disable

	// Pool sizing & lifecycle (sane defaults applied if zero)
	MaxConns          int32         // default 10
	MinConns          int32         // default 0
	HealthCheckPeriod time.Duration // default 30s
	MaxConnLifetime   time.Duration // default 0 (no limit)
	MaxConnIdleTime   time.Duration // default 5m
	DialTimeout       time.Duration // default 5s
}

// NewPool builds a pgxpool.Pool with sane defaults and verifies connectivity.
func NewPool(ctx context.Context, opt Options) (*pgxpool.Pool, error) {
	if opt.DatabaseURL == "" {
		return nil, fmt.Errorf("pgxkit: DatabaseURL is required")
	}
	cfg, err := pgxpool.ParseConfig(opt.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxkit: parse url: %w", err)
	}

	// Apply defaults
	if opt.MaxConns <= 0 {
		opt.MaxConns = 10
	}
	if opt.HealthCheckPeriod == 0 {
		opt.HealthCheckPeriod = 30 * time.Second
	}
	if opt.MaxConnIdleTime == 0 {
		opt.MaxConnIdleTime = 5 * time.Minute
	}
	if opt.DialTimeout == 0 {
		opt.DialTimeout = 5 * time.Second
	}

	cfg.MaxConns = opt.MaxConns
	cfg.MinConns = opt.MinConns
	cfg.HealthCheckPeriod = opt.HealthCheckPeriod
	cfg.MaxConnLifetime = opt.MaxConnLifetime
	cfg.MaxConnIdleTime = opt.MaxConnIdleTime
	// Use context timeouts on queries; network dial timeout is handled by the driver.

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxkit: new pool: %w", err)
	}

	// Verify connectivity quickly
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxkit: ping: %w", err)
	}
	return pool, nil
}
