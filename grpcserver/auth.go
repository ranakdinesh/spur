package grpcserver

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func authInterceptor(opt Options) grpc.UnaryServerInterceptor {
	if opt.ValidateToken == nil {
		// Auth disabled
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
			return h(ctx, req)
		}
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
		// Allow unauthenticated methods (health, reflection, etc.)
		if isAllowlisted(info.FullMethod, opt.AllowAuthenticated) {
			return h(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		token := extractToken(md, opt.APIKeyHeader)
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}

		claims, err := opt.ValidateToken(token)
		if err != nil {
			// Respect status codes if the validator returns one
			if st, ok := status.FromError(err); ok {
				return nil, st.Err()
			}
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}

		if opt.EnrichContext != nil {
			ctx = opt.EnrichContext(ctx, token, claims)
		}
		return h(ctx, req)
	}
}

func extractToken(md metadata.MD, apiKeyHeader string) string {
	if vals := md.Get("authorization"); len(vals) > 0 {
		authz := vals[0]
		low := strings.ToLower(authz)
		if strings.HasPrefix(low, "bearer ") {
			return strings.TrimSpace(authz[7:])
		}
		// If a raw token was sent in Authorization without Bearer prefix, accept it
		return strings.TrimSpace(authz)
	}
	if apiKeyHeader != "" {
		if vals := md.Get(strings.ToLower(apiKeyHeader)); len(vals) > 0 {
			return strings.TrimSpace(vals[0])
		}
	}
	return ""
}

func isAllowlisted(fullMethod string, allow []string) bool {
	if len(allow) == 0 {
		return false
	}
	for _, a := range allow {
		if a == fullMethod || (strings.HasSuffix(a, "*") && strings.HasPrefix(fullMethod, strings.TrimSuffix(a, "*"))) {
			return true
		}
	}
	return false
}
