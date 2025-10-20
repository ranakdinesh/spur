package authclient

import (
	"context"
	"strings"

	"github.com/ranakdinesh/spur/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryServerAuth validates Authorization metadata and enriches ctx.
func UnaryServerAuth(v *Validator, opt Options, log *logger.Loggerx) grpc.UnaryServerInterceptor {
	requireJWT := v != nil

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
		// API key path
		if opt.APIKeyHeader != "" && opt.APIKeyValue != "" {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				if vals := md.Get(opt.APIKeyHeader); len(vals) > 0 && vals[0] == opt.APIKeyValue {
					return h(ctx, req)
				}
			}
		}

		// JWT path
		if !requireJWT {
			return nil, status.Error(16 /* Unauthenticated */, "unauthorized")
		}
		md, _ := metadata.FromIncomingContext(ctx)
		var raw string
		if md != nil {
			if vals := md.Get("authorization"); len(vals) > 0 {
				raw = vals[0]
			}
		}
		if raw == "" || !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			return nil, status.Error(16, "missing bearer token")
		}
		token := strings.TrimSpace(raw[len("Bearer "):])
		cl, err := v.Validate(ctx, token)
		if err != nil {
			log.Warn(ctx).Err(err).Msg("jwt validation failed")
			return nil, status.Error(16, "unauthorized")
		}
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
		return h(ctx, req)
	}
}

// UnaryClientPropagate adds Authorization and X-Request-Id to outbound calls from ctx.
func UnaryClientPropagate() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, _ := metadata.FromOutgoingContext(ctx)
		out := md.Copy()
		if tok, ok := TokenFrom(ctx); ok && tok != "" {
			out.Set("authorization", "Bearer "+tok)
		}
		if rid, ok := logger.TraceIDFrom(ctx); ok && rid != "" {
			out.Set("x-request-id", rid)
		}
		ctx = metadata.NewOutgoingContext(ctx, out)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
