package grpcserver

import (
    "fmt"
    "net"

    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"

    "github.com/ranakdinesh/spur/config"
    "github.com/ranakdinesh/spur/logger"
)

type GRPCServerX struct {
    cfg    *config.Config
    log    *logger.Loggerx
    server *grpc.Server
    lis    net.Listener
    health *health.Server
}

// New constructs a gRPC server with health service registered.
func New(cfg *config.Config, log *logger.Loggerx) (*GRPCServerX, error) {
    gs := grpc.NewServer()
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
    if err != nil { return nil, err }

    hs := health.NewServer()
    healthpb.RegisterHealthServer(gs, hs)

    return &GRPCServerX{cfg: cfg, log: log, server: gs, lis: lis, health: hs}, nil
}

// Register allows parent application to register their protobuf services.
func (g *GRPCServerX) Register(reg func(s *grpc.Server)) {
    reg(g.server)
}

func (g *GRPCServerX) Start() error {
    g.log.Logger.Info().Int("port", g.cfg.GRPC.Port).Msg("grpc server listening")
    return g.server.Serve(g.lis)
}

func (g *GRPCServerX) Stop() {
    g.log.Logger.Info().Msg("stopping grpc server")
    g.health.Shutdown()
    g.server.GracefulStop()
}
