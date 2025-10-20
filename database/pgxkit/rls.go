package pgxkit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenantRole sets RLS session variables (SET LOCAL app.tenant_id/app.role) for the duration
// of a transaction and runs fn. Use this when your Postgres policies rely on these vars.
func WithTenantRole(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, role string, fn func(ctx context.Context, tx pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("pgxkit: nil pool")
	}
	return pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL app.tenant_id = $1`, tenantID); err != nil {
			return fmt.Errorf("pgxkit: set tenant_id: %w", err)
		}
		if _, err := tx.Exec(ctx, `SET LOCAL app.role = $1`, role); err != nil {
			return fmt.Errorf("pgxkit: set role: %w", err)
		}
		return fn(ctx, tx)
	})
}
