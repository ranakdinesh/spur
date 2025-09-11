package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ranakdinesh/spur/config"
	"github.com/ranakdinesh/spur/logger"
)

type Postgres struct {
	Pool *pgxpool.Pool
	log  *logger.Loggerx
}

func New(ctx context.Context, cfg *config.Config, log *logger.Loggerx) (*Postgres, error) {
	conf, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return nil, err
	}
	conf.MaxConns = 8
	conf.MinConns = 1
	conf.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	return &Postgres{Pool: pool, log: log}, nil
}

func (p *Postgres) Close() { p.Pool.Close() }

// Exec executes a statement; args are encoded by pgx natively.
func (p *Postgres) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.Pool.Exec(ctx, sql, args...)
}

func QueryMapped[T any](ctx context.Context, p *Postgres, sql string, mapper func(pgx.Row) (T, error), args ...any) ([]T, error) {
	rows, err := p.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]T, 0)
	for rows.Next() {
		v, err := mapper(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func QueryRowMapped[T any](ctx context.Context, p *Postgres, sql string, scan func(pgx.Row) (T, error), args ...any) (T, error) {
	r := p.Pool.QueryRow(ctx, sql, args...)
	return scan(r)
}
