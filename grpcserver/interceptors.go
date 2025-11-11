package grpcserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/ranakdinesh/spur/logger"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const headerRequestID = "x-request-id"

func buildUnaryChain(opt Options) grpc.UnaryServerInterceptor {
	var chain []grpc.UnaryServerInterceptor
	if opt.EnableReqID {
		chain = append(chain, reqIDInterceptor())
	}
	if opt.RateLimit != nil {
		chain = append(chain, rateLimitInterceptor(opt.RateLimit))
	}
	if opt.ValidateToken != nil {
		chain = append(chain, authInterceptor(opt))
	}
	if opt.EnableRecovery {
		chain = append(chain, recoveryInterceptor(opt.Log))
	}
	if opt.EnableAccessLogs {
		chain = append(chain, accessLogInterceptor(opt.Log))
	}
	return chainUnary(chain...)
}
func chainUnary(inters ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	switch len(inters) {
	case 0:
		return nil
	case 1:
		return inters[0]
	default:
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			// fold from right to left
			h := handler
			for i := len(inters) - 1; i >= 0; i-- {
				next := h
				inter := inters[i]
				h = func(c context.Context, r interface{}) (interface{}, error) {
					return inter(c, r, info, next)
				}
			}
			return h(ctx, req)
		}
	}
}

func reqIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		var rid string
		if md != nil {
			if v := md.Get(headerRequestID); len(v) > 0 && v[0] != "" {
				rid = v[0]
			}
		}
		if rid == "" {
			// basic unique-ish id
			rid = fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand16())
			// add to context metadata for downstream propagation
			mdOut := metadata.Pairs(headerRequestID, rid)
			ctx = metadata.NewIncomingContext(ctx, metadata.Join(md, mdOut))
		}
		// also inject into logger context if available
		if lctx := logger.WithTraceID(ctx, rid); lctx != nil {
			ctx = lctx
		}
		return handler(ctx, req)
	}
}

func accessLogInterceptor(log *logger.Loggerx) grpc.UnaryServerInterceptor {
	if log == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)
		log.Info(ctx).Str("grpc_method", info.FullMethod).
			Str("code", code.String()).
			Float64("duration_ms", float64(time.Since(start).Microseconds())/1e3).
			Msg("grpc request")
		return resp, err
	}
}

func recoveryInterceptor(log *logger.Loggerx) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprintf("panic: %v", r)
				if log != nil {
					log.Error(ctx).Msg(msg)
				}
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

func rateLimitInterceptor(o *RateLimitOptions) grpc.UnaryServerInterceptor {
	if o == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}
	var lim *rate.Limiter
	if o.Limiter != nil {
		lim = o.Limiter
	} else {
		rps := o.RPS
		if rps <= 0 {
			return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				return handler(ctx, req)
			}
		}
		burst := o.Burst
		if burst <= 0 {
			burst = 10
		}
		lim = rate.NewLimiter(rate.Limit(rps), burst)
	}
	keyMode := strings.ToLower(o.Key)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if keyMode == "ip" {
			// Best-effort: use peer address in ctx; fallback to global limiter.
			// In a real impl you'd maintain a map[ip]*Limiter with cleanup.
		}
		if !lim.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}

// helpers

var _seed uint64 = 0x9e3779b97f4a7c15

func rand16() uint16 {
	_seed ^= _seed << 13
	_seed ^= _seed >> 7
	_seed ^= _seed << 17
	return uint16((_seed >> 8) & 0xffff)
}

// RemoteIP is a tiny helper to parse host:port.
func RemoteIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// drainAndClose is handy if you add stream interceptors later.
func drainAndClose(c io.Closer) {
	_ = c.Close()
}
