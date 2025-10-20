package otelx

import (
	"net/http"

	"github.com/ranakdinesh/spur/logger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Middleware wraps handlers with otelhttp and injects trace_id into ctx for logger.
func Middleware(service string, l *logger.Loggerx) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// After otelhttp starts span, it’s already in the context.
			// We grab SpanContext to push trace_id into logger ctx.
			span := trace.SpanFromContext(r.Context())
			sc := span.SpanContext()
			ctx := r.Context()
			if sc.IsValid() {
				ctx = logger.WithTraceID(ctx, sc.TraceID().String())
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
		return otelhttp.NewHandler(h, service)
	}
}

// Transport wraps any RoundTripper so outbound HTTP gets spans + propagation.
func Transport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(base)
}
