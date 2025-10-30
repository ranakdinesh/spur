package authclient

import "time"

type Options struct {
	// JWT/JWKS validation
	Issuer   string   // expected iss
	Audience []string // accepted aud
	JWKSURL  string   // https://issuer/.well-known/jwks.json

	// Caching/refresh
	RefreshInterval time.Duration // default 10m
	RefreshTimeout  time.Duration // default 2s

	// AuthZ (optional)
	RequiredScopes []string // if non-empty, token must contain all

	// API key (optional, for service-to-service where JWT not used)
	APIKeyHeader string // e.g. "X-API-Key"
	APIKeyValue  string // if set, inbound must match this exact value
}
