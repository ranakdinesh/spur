package grpcserver

import (
	"github.com/ranakdinesh/spur/logger"
	"golang.org/x/time/rate"
)

// Options controls gRPC server behavior.
type Options struct {
	Addr string // e.g. ":9090"

	// Toggles
	EnableReflection bool // typically true in dev, false in prod
	EnableHealth     bool // registers standard gRPC health service

	// Middlewares
	EnableRecovery   bool // panic → internal error, logs stack
	EnableReqID      bool // attach/propagate x-request-id
	EnableAccessLogs bool // structured logs for each RPC

	// Auth & RBAC hooks (optional)
	AuthFunc UnaryAuthFunc // if set, called on each unary request

	// Rate limiting (optional)
	UnaryRateLimiter *rate.Limiter // nil = off

	// Dependencies
	Log *logger.Loggerx // required
}

// UnaryAuthFunc lets you plug simple auth without wiring a full dependency.
type UnaryAuthFunc func(fullMethod string, claims map[string]any) error

// RegisterFunc allows parent apps to register their services.
type RegisterFunc func(s GRPCRegistrar)

// GRPCRegistrar is the subset of *grpc.Server we need for registration.
// It keeps your parent code testable by accepting mocks/fakes if needed.
type GRPCRegistrar interface {
	RegisterService(desc *ServiceDesc, impl interface{})
}

// We mirror minimal bits from google.golang.org/grpc to decouple this package
type ServiceDesc = struct {
	ServiceName string
	HandlerType interface{}
	Methods     []MethodDesc
	Streams     []StreamDesc
	Metadata    interface{}
}
type MethodDesc = struct {
	MethodName string
	Handler    interface{}
}
type StreamDesc = struct {
	StreamName    string
	Handler       interface{}
	ServerStreams bool
	ClientStreams bool
}
