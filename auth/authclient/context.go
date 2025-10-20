package authclient

import "context"

type ctxKey string

const (
	ctxToken    ctxKey = "auth.token"
	ctxSubject  ctxKey = "auth.sub"
	ctxTenantID ctxKey = "auth.tenant_id"
	ctxUserID   ctxKey = "auth.user_id"
	ctxScopes   ctxKey = "auth.scopes"
)

func WithToken(ctx context.Context, tok string) context.Context {
	return context.WithValue(ctx, ctxToken, tok)
}
func TokenFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxToken).(string)
	return v, ok
}

func WithSubject(ctx context.Context, sub string) context.Context {
	return context.WithValue(ctx, ctxSubject, sub)
}
func SubjectFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxSubject).(string)
	return v, ok
}

func WithTenantID(ctx context.Context, tid string) context.Context {
	return context.WithValue(ctx, ctxTenantID, tid)
}
func TenantIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxTenantID).(string)
	return v, ok
}

func WithUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, ctxUserID, uid)
}
func UserIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserID).(string)
	return v, ok
}

func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, ctxScopes, scopes)
}
func ScopesFrom(ctx context.Context) ([]string, bool) {
	v, ok := ctx.Value(ctxScopes).([]string)
	return v, ok
}
