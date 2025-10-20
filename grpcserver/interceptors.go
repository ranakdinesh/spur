package grpcserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/ranakdinesh/spur/logger"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const mdKeyRequestID = "x-request-id"
const mdKeyAuth = "authorization"

// chainUnary creates a single interceptor from multiple.
func chainUnary(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	// remove nils
	var list []grpc.UnaryServerInterceptor
	for _, it := range interceptors {
		if it != nil {
			list = append(list, it)
		}
	}
	if len(list) == 0 {
		return nil
	}
	if len(list) == 1 {
		return list[0]
	}
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// build nested chain
		var h grpc.UnaryHandler = handler
		for i := len(list) - 1; i >= 0; i-- {
			curr := list[i]
			next := h
			h = func(c context.Context, r interface{}) (interface{}, error) {
				return curr(c, r, info, next)
			}
		}
		return h(ctx, req)
	}
}

// reqIDInterceptor ensures there's a request id in metadata and context.
func reqIDInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		var rid string
		if md != nil {
			vals := md.Get(mdKeyRequestID)
			if len(vals) > 0 && vals[0] != "" {
				rid = vals[0]
			}
		}
		if rid == "" {
			rid = newReqID()
			// propagate downstream
			mdOut := metadata.Pairs(mdKeyRequestID, rid)
			ctx = metadata.NewIncomingContext(ctx, metadata.Join(md, mdOut))
		}
		ctx = logger.WithTraceID(ctx, rid)
		return handler(ctx, req)
	}
}

func newReqID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// accessLogInterceptor logs method, code, latency, size if available.
func accessLogInterceptor(l *logger.Loggerx) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		dur := time.Since(start)

		st, _ := status.FromError(err)
		code := "OK"
		if st != nil {
			code = st.Code().String()
		}

		l.Info(ctx).
			Str("rpc", info.FullMethod).
			Str("code", code).
			Dur("latency", dur).
			Msg("grpc_request")

		return resp, err
	}
}

// recoveryInterceptor catches panics and converts to internal error, logging the stack.
func recoveryInterceptor(l *logger.Loggerx) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				l.Error(ctx).Interface("panic", r).Str("rpc", info.FullMethod).Msg("panic recovered")
				err = statusInternal() // generic
			}
		}()
		return handler(ctx, req)
	}
}

func statusInternal() error {
	// import kept local to avoid bleeding status codes everywhere
	return status.Error(13 /* codes.Internal */, "internal error")
}

// authInterceptor calls your provided AuthFunc if set.
// It parses "authorization: Bearer <token>" or "authorization: ApiKey <key>" into claims map.
func authInterceptor(auth UnaryAuthFunc) grpc.UnaryServerInterceptor {
	if auth == nil {
		return nil
	}
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		var token string
		if md != nil {
			if vals := md.Get(mdKeyAuth); len(vals) > 0 {
				token = vals[0]
			}
		}
		claims := map[string]any{}
		if token != "" {
			// very light parsing for pattern: "Bearer x" or "ApiKey y"
			parts := strings.SplitN(token, " ", 2)
			if len(parts) == 2 {
				claims["scheme"] = parts[0]
				claims["token"] = parts[1]
			} else {
				claims["token"] = token
			}
		}
		if err := auth(info.FullMethod, claims); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// rateLimitInterceptor gates requests via a token bucket.
func rateLimitInterceptor(lim interface{}) grpc.UnaryServerInterceptor {
	if lim == nil {
		return nil
	}
	rl, ok := lim.(*rate.Limiter)
	if !ok {
		return nil
	}
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !rl.Allow() {
			return nil, status.Error(8 /* ResourceExhausted */, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
