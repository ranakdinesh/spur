package grpcserver

import (
	"github.com/ranakdinesh/spur/logger"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

// Options controls gRPC server behavior.
type Options struct {
	Addr string // e.g. ":9090"

	// Toggles
	EnableReflection bool // typically true in dev, false in prod
	EnableHealth     bool // registers standard gRPC health service
	EnableReqID      bool // injects/propagates x-request-id
	EnableAccessLogs bool // structured access logs
	EnableRecovery   bool // panic recovery with error conversion

	// Rate limiting (optional)
	RateLimit *RateLimitOptions

	// Simple auth hook (optional)
	Auth UnaryAuthFunc

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
