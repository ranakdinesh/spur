package grpcclient

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/ranakdinesh/spur/auth/authclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type DialResult struct {
	Conn *grpc.ClientConn
}

// New dials a gRPC connection with sensible defaults and propagation interceptor.
// Call result.Conn.Close() on shutdown.
func New(ctx context.Context, opt Options) (*DialResult, error) {
	if opt.ConnectTimeout == 0 {
		opt.ConnectTimeout = 3 * time.Second
	}
	if opt.PerRPCTimeout == 0 {
		opt.PerRPCTimeout = 5 * time.Second
	}
	if opt.Backoff == (backoff.Config{}) {
		opt.Backoff = backoff.Config{
			BaseDelay:  100 * time.Millisecond,
			Multiplier: 1.6,
			Jitter:     0.2,
			MaxDelay:   2 * time.Second,
		}
	}
	if opt.Keepalive == (keepalive.ClientParameters{}) {
		opt.Keepalive = keepalive.ClientParameters{
			Time:                30 * time.Second, // send pings every 30s if no activity
			Timeout:             3 * time.Second,  // wait 3s for ping ack
			PermitWithoutStream: true,
		}
	}
	if opt.EnableRetries && opt.MaxAttempts == 0 {
		opt.MaxAttempts = 3
	}

	// Credentials
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	if !opt.Insecure {
		tc := opt.TLS
		if tc == nil {
			tc = &tls.Config{} // system roots
		}
		creds = grpc.WithTransportCredentials(credentials.NewTLS(tc))
	}

	// Service config for retries & per-RPC timeout
	// (grpc-go uses JSON; per-RPC timeout is set by client via context, but we also
	// include a default timeout policy here to be explicit.)
	sc := `{
	  "methodConfig": [{
	    "name": [{"service": ""}],
	    "timeout": "` + opt.PerRPCTimeout.String() + `"
	  }]
	}`
	if opt.EnableRetries {
		// Attach a simple retry policy for idempotent-ish codes.
		// Note: Server must be configured to allow retries in its service config for advanced behaviors.
		sc = `{
		  "methodConfig": [{
		    "name": [{"service": ""}],
		    "timeout": "` + opt.PerRPCTimeout.String() + `",
		    "retryPolicy": {
		      "MaxAttempts": ` + itoa(opt.MaxAttempts) + `,
		      "InitialBackoff": "0.1s",
		      "MaxBackoff": "2s",
		      "BackoffMultiplier": 1.6,
		      "RetryableStatusCodes": ["UNAVAILABLE","RESOURCE_EXHAUSTED","ABORTED"]
		    }
		  }]
		}`
	}

	// Interceptors: always include propagation
	unaries := []grpc.UnaryClientInterceptor{
		authclient.UnaryClientPropagate(),
	}

	// Append any extras (typed) if provided via adapter
	for _, e := range adaptExtra(opt.ExtraUnary) {
		unaries = append(unaries, e)
	}

	dialCtx, cancel := context.WithTimeout(ctx, opt.ConnectTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		opt.Target,
		creds,
		grpc.WithKeepaliveParams(opt.Keepalive),
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: opt.Backoff, MinConnectTimeout: opt.ConnectTimeout}),
		grpc.WithUnaryInterceptor(chainUnary(unaries...)),
		grpc.WithDefaultServiceConfig(sc),
		grpc.WithBlock(), // wait until connected (or timeout)
	)
	if err != nil {
		return nil, err
	}
	return &DialResult{Conn: conn}, nil
}

func chainUnary(list ...grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	if len(list) == 0 {
		return nil
	}
	if len(list) == 1 {
		return list[0]
	}
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Build nested chain
		var h grpc.UnaryInvoker = invoker
		for i := len(list) - 1; i >= 0; i-- {
			curr := list[i]
			next := h
			h = func(c context.Context, m string, r, rp interface{}, cn *grpc.ClientConn, op ...grpc.CallOption) error {
				return curr(c, m, r, rp, cn, next, op...)
			}
		}
		return h(ctx, method, req, reply, cc, opts...)
	}
}

// adaptExtra converts the loose field to real grpc interceptors if you choose to pass any.
func adaptExtra(_ []UnaryInterceptor) []grpc.UnaryClientInterceptor {
	// Keep simple for now; you can extend this to accept typed interceptors directly if you want.
	return nil
}

// tiny itoa to avoid fmt overhead on hot path (not critical, just neat)
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	buf := [24]byte{}
	pos := len(buf)
	n := i
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
