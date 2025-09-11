package spur

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ranakdinesh/spur/config"
	psql "github.com/ranakdinesh/spur/database/postgres"
	rds "github.com/ranakdinesh/spur/database/redis"
	"github.com/ranakdinesh/spur/grpcserver"
	"github.com/ranakdinesh/spur/httpserver"
	"github.com/ranakdinesh/spur/logger"
)

type Spur struct {
	Cfg      *config.Config
	Log      *logger.Loggerx
	HTTP     *httpserver.HTTPServerX
	GRPC     *grpcserver.GRPCServerX
	Postgres *psql.Postgres
	Redis    *rds.Redis

	// controls
	useHTTP     bool
	useGRPC     bool
	usePostgres bool
	useRedis    bool

	wg sync.WaitGroup
}

// New initializes Spur and its selected subsystems.
func New(ctx context.Context, cfg *config.Config, opts ...Option) (*Spur, error) {
	if cfg == nil {
		c, err := config.Load()
		if err != nil {
			return nil, err
		}
		cfg = c
	}

	// Logger
	lg := logger.New(cfg)

	s := &Spur{
		Cfg:     cfg,
		Log:     lg,
		useHTTP: cfg.HTTP.Enable,
		useGRPC: cfg.GRPC.Enable,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.usePostgres {
		// DBs (lazy errors permitted; services may not need them)
		if pg, err := psql.New(ctx, cfg, lg); err == nil {
			s.Postgres = pg
		} else {
			lg.Logger.Warn().Err(err).Msg("postgres init failed (continuing)")
		}
	}
	if s.useRedis {
		if rd, err := rds.New(ctx, cfg, lg); err == nil {
			s.Redis = rd
		} else {
			lg.Logger.Warn().Err(err).Msg("redis init failed (continuing)")
		}
	}
	// Servers
	if s.useHTTP {
		srv, err := httpserver.New(cfg, lg)
		if err != nil {
			return nil, err
		}
		s.HTTP = srv
	}
	if s.useGRPC {
		gsrv, err := grpcserver.New(cfg, lg)
		if err != nil {
			return nil, err
		}
		s.GRPC = gsrv
	}

	return s, nil
}

// Option for Spur
type Option func(*Spur)

func WithHTTP(enabled bool) Option     { return func(s *Spur) { s.useHTTP = enabled } }
func WithGRPC(enabled bool) Option     { return func(s *Spur) { s.useGRPC = enabled } }
func WithPostgres(enabled bool) Option { return func(s *Spur) { s.usePostgres = enabled } }
func WithRedis(enabled bool) Option    { return func(s *Spur) { s.useRedis = enabled } }

// Start all enabled servers and block until signal, then shutdown gracefully.
func (s *Spur) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	// Start servers
	if s.HTTP != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.HTTP.Start(); err != nil {
				s.Log.Logger.Error().Err(err).Msg("http server stopped with error")
				cancel()
			}
		}()
	}
	if s.GRPC != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.GRPC.Start(); err != nil {
				s.Log.Logger.Error().Err(err).Msg("grpc server stopped with error")
				cancel()
			}
		}()
	}

	select {
	case <-ctx.Done():
	case sig := <-sigc:
		s.Log.Logger.Info().Str("signal", sig.String()).Msg("shutdown requested")
	}

	// Graceful stop
	shutCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	s.Shutdown(shutCtx)
	s.wg.Wait()
	s.Log.Logger.Info().Msg("all services stopped")
	return nil
}

func (s *Spur) Shutdown(ctx context.Context) {
	if s.GRPC != nil {
		s.GRPC.Stop()
	}
	if s.HTTP != nil {
		_ = s.HTTP.Stop(ctx)
	}
	if s.Postgres != nil {
		s.Postgres.Close()
	}
	if s.Redis != nil {
		s.Redis.Close()
	}
}

func (s *Spur) String() string {
	return fmt.Sprintf("Spur{HTTP:%v GRPC:%v}", s.useHTTP, s.useGRPC)
}
