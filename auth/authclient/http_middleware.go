package authclient

import (
	"net/http"
	"strings"

	"github.com/ranakdinesh/spur/logger"
)

// HTTPAuth enforces Bearer JWT or API key (if configured). On success it enriches context.
func HTTPAuth(v *Validator, opt Options, log *logger.Loggerx) func(http.Handler) http.Handler {
	requireJWT := v != nil

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// API key path (optional)
			if opt.APIKeyHeader != "" && opt.APIKeyValue != "" {
				if val := r.Header.Get(opt.APIKeyHeader); val == opt.APIKeyValue {
					next.ServeHTTP(w, r)
					return
				}
			}
			// JWT path
			if !requireJWT {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			raw := r.Header.Get("Authorization")
			if raw == "" || !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(raw[len("Bearer "):])

			cl, err := v.Validate(r.Context(), token)
			if err != nil {
				log.Warn(r.Context()).Err(err).Msg("jwt validation failed")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = WithToken(ctx, token)
			if cl.Subject != "" {
				ctx = WithSubject(ctx, cl.Subject)
				ctx = logger.WithUserID(ctx, cl.Subject)
			}
			if cl.TenantID != "" {
				ctx = WithTenantID(ctx, cl.TenantID)
				ctx = logger.WithTenantID(ctx, cl.TenantID)
			}
			if len(cl.Scope) > 0 {
				ctx = WithScopes(ctx, cl.Scope)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RoundTripper that injects Authorization (JWT) and X-Request-Id if present in ctx.
type AuthTransport struct{ Base http.RoundTripper }

func (t AuthTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt := t.Base
	if rt == nil {
		rt = http.DefaultTransport
	}
	if tok, ok := TokenFrom(r.Context()); ok && tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	if rid, ok := logger.TraceIDFrom(r.Context()); ok && rid != "" {
		r.Header.Set("X-Request-Id", rid)
	}
	return rt.RoundTrip(r)
}
