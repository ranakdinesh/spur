package grpcserver

import (
	"context"
	"github.com/ranakdinesh/spur/logger"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

type Claims struct {
	Subject  string
	TenantID string
	Scope    []string
	Raw      map[string]any
}

type ValidateTokenFunc func(token string) (*Claims, error)

type ContextEnricher func(ctx context.Context, token string, c Claims) context.Context

// Options controls gRPC server behavior.
type Options struct {
	Addr string // e.g. ":9090"

	// Toggles
	EnableReflection bool // typically true in dev, false in prod
	EnableHealth     bool // registers standard gRPC health service
	EnableReqID      bool // injects/propagates x-request-id
	EnableAccessLogs bool // structured access logs
	EnableRecovery   bool // panic recovery with error conversion

	// Authentication: when validateToken is not nil, auth is enforced by default
	ValidateToken      ValidateTokenFunc
	APIKeyHeader       string          // optional alternative header
	AllowAuthenticated []string        // full method names or prefixes allowed without auth
	EnrichContext      ContextEnricher // optional context enrichment hook

	// Rate limiting (optional)
	RateLimit *RateLimitOptions

	// Simple auth hook (optional)
	//Auth UnaryAuthFunc

	// Dependencies
	Log            *logger.Loggerx // required
	TracerProvider trace.TracerProvider
}

// RateLimitOptions configures a token-bucket limiter.
type RateLimitOptions struct {
	// Requests per second (tokens per second). If 0, rate limit is disabled.
	RPS float64
	// Burst controls maximum burst size for the limiter.
	Burst int
	// Key is optional; if provided "global" for single limiter, or "ip" for per-remote IP.
	// Defaults to "global".
	Key string
	// Optional custom limiter. If provided, RPS/Burst/Key are ignored.
	Limiter *rate.Limiter
}

// UnaryAuthFunc lets you plug simple auth without wiring a full dependency.
type UnaryAuthFunc func(fullMethod string, claims map[string]any) error

// RegisterFunc allows parent apps to register their services.
type RegisterFunc func(s GRPCRegistrar)
