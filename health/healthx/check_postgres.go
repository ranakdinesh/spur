package healthx

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgCheck struct{ p *pgxpool.Pool }

func (pgCheck) Name() string { return "postgres" }

func (c pgCheck) Check(ctx context.Context) error {
	if c.p == nil {
		return errors.New("nil pool")
	}
	return c.p.Ping(ctx)
}

func Postgres(p *pgxpool.Pool) Checker { return pgCheck{p: p} }
