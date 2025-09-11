# Spur (Multipackage)

Production-ready starter for Go microservices:

- HTTP server (Chi) with health middleware, route groups, optional CORS
- gRPC server with graceful lifecycle
- Structured logging via zerolog (dev: console; prod: forward to logger-service placeholder)
- Postgres (pgxpool) + Redis clients
- Utilities: files, directories, crypto, password, slug, JSON, uploads/downloads

## Quick start

```bash
# inside this folder
go mod tidy
go run ./cmd/example
```

The example reads env from `.env` (fallback defaults). Press Ctrl+C to stop gracefully.
