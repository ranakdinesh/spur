package pgxkit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTx wraps a function in a BEGIN/COMMIT (or ROLLBACK on error).
// Use for unit-of-work; keep work inside the fn fast and bounded.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context, tx pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("pgxkit: nil pool")
	}
	return pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return fn(ctx, tx)
	})
}
