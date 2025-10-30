package grpcserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// authInterceptor bridges the pluggable UnaryAuthFunc into a gRPC unary interceptor.
// It extracts credentials from incoming metadata and provides a lightweight claims map.
//
// Notes:
//   - If an "authorization: Bearer <JWT>" header is present, the JWT payload is decoded
//     WITHOUT signature verification (for convenience). Your UnaryAuthFunc should
//     perform real verification/validation (e.g., using your authclient.Validator).
//   - If you prefer to do all work via ctx/metadata, ignore the claims map and read from ctx.
func authInterceptor(fn UnaryAuthFunc) grpc.UnaryServerInterceptor {
	if fn == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		claims := extractClaimsFromMD(ctx)
		if err := fn(info.FullMethod, claims); err != nil {
			// Map any error from your auth function to PermissionDenied.
			// If you want different codes, return a status.Error from fn and detect it here.
			st, ok := status.FromError(err)
			if ok {
				return nil, st.Err()
			}
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return handler(ctx, req)
	}
}

func extractClaimsFromMD(ctx context.Context) map[string]any {
	md, _ := metadata.FromIncomingContext(ctx)
	if md == nil {
		return nil
	}
	// Try Authorization: Bearer <jwt>
	if vals := md.Get("authorization"); len(vals) > 0 {
		token := vals[0]
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			jwt := strings.TrimSpace(token[len("bearer "):])
			if m := decodeJWTClaims(jwt); m != nil {
				return m
			}
		}
	}
	// Fallback: if someone passed "x-claims" header with JSON
	if vals := md.Get("x-claims"); len(vals) > 0 {
		var m map[string]any
		if json.Unmarshal([]byte(vals[0]), &m) == nil {
			return m
		}
	}
	return nil
}

func decodeJWTClaims(jwt string) map[string]any {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil
	}
	payload := parts[1]
	// base64url decode with padding fix
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}
	b, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(b, &out) != nil {
		return nil
	}
	return out
}
