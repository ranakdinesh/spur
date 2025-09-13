package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/ranakdinesh/spur/config"
	"github.com/ranakdinesh/spur/logger"
)

// RouteGroup is a function that mounts a group of routes on a subrouter.
type RouteGroup func(r chi.Router)

type HTTPServerX struct {
	cfg    *config.Config
	log    *logger.Loggerx
	Router *chi.Mux
	server *http.Server
}

// New constructs the HTTP server with health middleware and optional CORS.
func New(cfg *config.Config, log *logger.Loggerx) (*HTTPServerX, error) {
	r := chi.NewRouter()

	// base middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Heartbeat("/healthz"))
	r.Use(middleware.Recoverer)

	// latency logger
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, req)
			log.Logger.Debug().
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Dur("latency", time.Since(start)).
				Msg("http_request")
		})
	})

	if cfg.HTTP.EnableCORS {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &HTTPServerX{
		cfg:    cfg,
		log:    log,
		Router: r,
		server: srv,
	}, nil
}

// MountGroup allows parent app to mount a group of routes at a path prefix.
func (s *HTTPServerX) MountGroup(prefix string, rg RouteGroup) {
	s.Router.Route(prefix, func(r chi.Router) {
		rg(r)
	})
}

// AddRoute allows mounting arbitrary handlers to the root router.
func (s *HTTPServerX) AddRoute(method, path string, handler http.HandlerFunc) {
	s.Router.Method(method, path, handler)
}

// Start begins listening.
func (s *HTTPServerX) Start() error {
	s.log.Info(context.Background(), "Http Server Started and Listening on ", map[string]interface{}{"port": s.cfg.HTTP.Port})

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *HTTPServerX) Stop(ctx context.Context) error {
	s.log.Logger.Info().Msg("stopping http server")
	return s.server.Shutdown(ctx)
}
