# spur

A modern Go starter framework for cloud‑native microservices.

Spur is a production-grade starter kit to help you bootstrap Go services quickly — with batteries included but no bloat. It provides opinionated building blocks for HTTP, gRPC, metrics, tracing, configuration, logging, database, caching, and health checks, while staying true to Go’s simplicity and composability.
---

## Table of Contents

- Key Features
- Quick Start
    - Install
    - Scaffold a service
    - Run locally
    - Enable metrics and tracing
- Package Overview
- Deploy to Kubernetes
- CI/CD
- Observability Integration
- Design Philosophy
- Example: Minimal HTTP Service
- Roadmap
- Contributing
- License

---

## Key Features

| Category | Features |
|---------|----------|
| Server | Unified HTTP, gRPC servers with pluggable middlewares |
| Configuration | Typed .env + environment loader (configx) |
| Logging | Structured Zerolog-based logs with optional remote sink |
| Auth | Reusable JWT/OIDC validator (authclient) |
| Databases | pgx PostgreSQL + Redis helper packages |
| Health & Readiness | Pluggable health checks with JSON endpoints |
| Metrics (opt-in) | Prometheus metrics via -tags=metrics |
| Tracing | OpenTelemetry support for HTTP + gRPC |
| CLI | spur new service scaffolds fully-wired microservices |
| Kubernetes-ready | Kustomize manifests, probes, ConfigMap + Secret patterns |

---

## Quick Start

### 1) Install Spur CLI

```bash
go install github.com/ranakdinesh/spur/cmd/spur@latest
```

### 2) Scaffold a new service

```bash
spur new service accounts \
  --module github.com/you/accounts \
  --with-db --with-redis --with-auth \
  --httpaddr :8081
```

This creates a new project structure:

```text
accounts/
├── cmd/accounts/main.go
├── internal/
│   ├── app/
│   └── ...
├── k8s/
│   ├── base/
│   └── overlays/
├── .github/workflows/
└── .env.example
```

### 3) Run locally

```bash
cd accounts
go run ./cmd/accounts
```

Default endpoints:

```bash
curl http://localhost:8081/health/live
curl http://localhost:8081/health/ready
```

### 4) (Optional) Enable metrics and tracing

```bash
go run -tags="metrics" ./cmd/accounts
```

If you have an OTLP collector:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

Spur automatically emits spans for HTTP and gRPC requests.

---

## Package Overview

### configx
Declarative environment loader with validation.

```go
package main

type AppConfig struct {
  AppEnv string `env:"APP_ENV" default:"development"`
  Port   int    `env:"PORT" default:"8080"`
  DBURL  string `env:"DATABASE_URL" required:"true"`
}

func main() {
  cfg := AppConfig{}
  configx.MustLoad(&cfg)
}
```

### logger
Structured Zerolog logger with optional remote sink.

```go
log := logger.NewWithOptions(logger.Options{
  Dev: true,
  EnableHTTPSink: false,
})
log.Info(context.Background()).Msg("server started")
```

### httpserver
Secure chi-based HTTP server.

```go
srv := httpserver.NewServer(httpserver.Options{
  Addr: ":8080",
  EnableCORS: true,
}, log, func(r chi.Router) {
  r.Get("/hello", func(w http.ResponseWriter, _ *http.Request) {
    w.Write([]byte("Hello Spur!"))
  })
})

srv.Start(context.Background())
```

### pgxkit and rediskit
Production-safe connection helpers with context management.

```go
pool, _ := pgxkit.NewPool(ctx, pgxkit.Options{DatabaseURL: cfg.DBURL})
rdb, _  := rediskit.NewClient(ctx, rediskit.Options{Addr: cfg.RedisAddr})
```

### authclient
JWT/OIDC validator with coreos/go-oidc under the hood.

```go
v, _ := authclient.NewValidator(ctx, authclient.Options{
  Issuer:   "https://issuer.example.com",
  Audience: []string{"my-service"},
})
claims, _ := v.Validate(ctx, token)
```

### healthx
Simple pluggable health checks.

```go
agg := healthx.New()
agg.Register(healthx.Postgres(pool), healthx.Redis(rdb))
healthx.Mount(router, agg)
```

### metricsx (optional, build with -tags=metrics)
Prometheus middleware and gRPC interceptors.

```go
reg := metricsx.NewRegistry()
httpm := metricsx.NewHTTPMetrics(reg, "accounts")
r.Use(httpm.Middleware)
r.Handle("/metrics", reg.Handler())
```

### otelx
OpenTelemetry setup for distributed tracing.

```go
shutdown, _ := otelx.Start(ctx, otelx.Options{
  ServiceName:  "accounts",
  OTLPEndpoint: "http://otel-collector:4318",
})
defer shutdown(ctx)
```

### grpcclient and grpcserver
Consistent gRPC dialer and server with propagation and retries.

```go
gc, _ := grpcclient.New(ctx, grpcclient.Options{Target: "localhost:9090", Insecure: true})
conn := gc.Conn
defer conn.Close()
```

---

## Deploy to Kubernetes

```bash
kubectl apply -k k8s/overlays/dev
# or for production
kubectl apply -k k8s/overlays/prod
```

Default health probes:

- /health/live
- /health/ready

---

## CI/CD

Each scaffolded service includes:

- CI: lint, test, build, push to GHCR
- Release: build on tagged commits
- Artifacts: ghcr.io/<org>/<service>:latest and ghcr.io/<org>/<service>:<git-sha>

---

## Observability Integration

| Tool | Integration |
|------|-------------|
| Prometheus | /metrics exposed via ServiceMonitor |
| Grafana | Ready to visualize spur_http_* and spur_grpc_* metrics |
| Jaeger / Tempo / OTEL | Use OTEL_EXPORTER_OTLP_ENDPOINT to export spans |
| Elasticsearch / Loki | Forward Zerolog JSON logs via your log agent |

---

## Design Philosophy

- Composable, not monolithic. Import only what you need.
- Production-first defaults: secure headers, health probes, timeouts, structured logs.
- Zero magic, full control: no hidden goroutines or reflection-heavy frameworks.
- Deploy anywhere: local Docker/k3d to Kubernetes.
- Solo-developer friendly: go from idea to deploy in minutes.

---

## Example: Minimal HTTP Service

```go
package main

import (
  "context"
  "net/http"

  "github.com/go-chi/chi/v5"
  "github.com/ranakdinesh/spur/config"
  "github.com/ranakdinesh/spur/httpserver"
  "github.com/ranakdinesh/spur/logger"
)

type Config struct {
  Addr string `env:"ADDR" default:":8080"`
}

func main() {
  cfg := Config{}
  configx.MustLoad(&cfg)
  log := logger.NewWithOptions(logger.Options{Dev: true})

  srv := httpserver.NewServer(httpserver.Options{Addr: cfg.Addr}, log, func(r chi.Router) {
    r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
      _, _ = w.Write([]byte("Hello Spur!"))
    })
  })

  log.Info(context.Background()).Msgf("starting server on %s", cfg.Addr)
  srv.Start(context.Background())
}
```

---

## Roadmap

- [x] Metrics and tracing
- [x] Health checks
- [x] Kubernetes templates
- [ ] Background job runner
- [ ] CLI plugin for microservice registry
- [ ] GraphQL and WebSocket helpers
- [ ] Code generation templates (SQLC + gRPC + docs)

---

## Contributing

1. Fork this repository
2. Create a branch: feature/awesome
3. Commit your changes
4. Open a pull request

All contributions are welcome — docs, bug fixes, ideas, even typos.

---

## License

MIT © 2025 Dinesh Rana


