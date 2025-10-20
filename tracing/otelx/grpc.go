package otelx

import (
	"context"

	"github.com/ranakdinesh/spur/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// ServerUnary returns an interceptor that both creates spans and injects trace_id for logger.
func ServerUnary(l *logger.Loggerx) grpc.UnaryServerInterceptor {
	otel := otelgrpc.UnaryServerInterceptor()
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
		resp, err := otel(ctx, req, info, func(c context.Context, r interface{}) (interface{}, error) {
			if sc := trace.SpanFromContext(c).SpanContext(); sc.IsValid() {
				c = logger.WithTraceID(c, sc.TraceID().String())
			}
			return h(c, r)
		})
		return resp, err
	}
}

// ClientUnary adds spans + propagation for outbound gRPC.
func ClientUnary() grpc.UnaryClientInterceptor {
	return otelgrpc.UnaryClientInterceptor()
}
