package grpcclient

import (
	"crypto/tls"
	"time"

	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"
)

type Options struct {
	// Target address, e.g. "dns:///users.svc.cluster.local:9090" or "localhost:9090"
	Target string

	// Security
	Insecure bool        // true = plaintext (dev only)
	TLS      *tls.Config // if set, used when Insecure=false

	// Keepalive (client-side pings to keep NAT/LB flows warm)
	Keepalive keepalive.ClientParameters // defaults applied if zero

	// Dial timeouts & backoff
	ConnectTimeout time.Duration  // default 3s
	Backoff        backoff.Config // default gRPC backoff (slightly tuned)

	// Retries via service-config (see RetryPolicy below)
	EnableRetries bool
	MaxAttempts   int           // default 3 (initial + 2 retries)
	Hedging       bool          // not used by default; reserved
	PerRPCTimeout time.Duration // default 5s (metadata "grpc-timeout")

	// Interceptors
	// Additional unary interceptors (e.g., metrics/tracing) are appended after propagation.
	ExtraUnary []UnaryInterceptor
}

// Minimal alias to avoid importing grpc in options.
type UnaryInterceptor func(ctx interface{}, method string, req, reply interface{}, cc interface{}, invoker interface{}, opts ...interface{}) error
