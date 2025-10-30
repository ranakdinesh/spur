package pgxkit

import "context"

type ctxKey string

const (
	ctxTenantID ctxKey = "tenant_id"
	ctxRole     ctxKey = "role"
)

func WithTenantInCtx(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ctxTenantID, tenantID)
}
func WithRoleInCtx(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ctxRole, role)
}
func TenantFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxTenantID).(string)
	return v, ok
}
func RoleFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxRole).(string)
	return v, ok
}
