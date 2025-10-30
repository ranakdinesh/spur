package grpcserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server wraps *grpc.Server with an address and Start method.
type Server struct {
	s    *grpc.Server
	addr string
}

func New(opt Options, register func(GRPCRegistrar)) (*Server, error) {
	var opts []grpc.ServerOption

	if uni := buildUnaryChain(opt); uni != nil {
		opts = append(opts, grpc.ChainUnaryInterceptor(uni))
	}

	gs := grpc.NewServer(opts...)

	if opt.EnableHealth {
		healthpb.RegisterHealthServer(gs, health.NewServer())
	}
	if opt.EnableReflection {
		reflection.Register(gs)
	}

	if register != nil {
		register(gs)
	}

	return &Server{s: gs, addr: opt.Addr}, nil
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		s.s.GracefulStop()
	}()
	return s.s.Serve(lis)
}
