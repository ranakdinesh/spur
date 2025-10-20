// auth/authclient/validator.go
package authclient

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
)

// Claims stays the same so the rest of your code doesn't change.
type Claims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tenant_id,omitempty"`
	UserID   string   `json:"user_id,omitempty"`
	Scope    []string `json:"scope,omitempty"`
	Scopes   string   `json:"scopes,omitempty"`
}

type Validator struct {
	opt      Options
	verifier *oidc.IDTokenVerifier
}

// NewValidator creates an OIDC verifier. Prefer discovery via Issuer,
// or set JWKSURL explicitly if your provider doesn't expose discovery.
func NewValidator(ctx context.Context, opt Options) (*Validator, error) {
	if opt.Issuer == "" && opt.JWKSURL == "" {
		return nil, errors.New("authclient: either Issuer or JWKSURL must be set")
	}

	var verifier *oidc.IDTokenVerifier
	cfg := &oidc.Config{
		ClientID: optAudienceFirst(opt.Audience),
		// You can set SupportedSigningAlgs if you want to restrict algs.
	}

	switch {
	case opt.Issuer != "":
		// Use OIDC discovery (preferred). This internally tracks the provider JWKS.
		p, err := oidc.NewProvider(ctx, opt.Issuer)
		if err != nil {
			return nil, err
		}
		verifier = p.Verifier(cfg) // caches keys & refreshes as needed. :contentReference[oaicite:2]{index=2}

	default:
		// No issuer? Fall back to a remote JWKS + explicit issuer (required by OIDC).
		if opt.Issuer == "" {
			return nil, errors.New("authclient: Issuer is required when using JWKSURL")
		}
		ks := oidc.NewRemoteKeySet(ctx, opt.JWKSURL)     // background-refreshing key set. :contentReference[oaicite:3]{index=3}
		verifier = oidc.NewVerifier(opt.Issuer, ks, cfg) // manual wiring (no discovery). :contentReference[oaicite:4]{index=4}
	}

	return &Validator{opt: opt, verifier: verifier}, nil
}

func (v *Validator) Close() { /* go-oidc manages its own background refresh; nothing to close */ }

func (v *Validator) Validate(ctx context.Context, tokenString string) (*Claims, error) {
	// Verify signature + iss/aud/time using go-oidc
	idt, err := v.verifier.Verify(ctx, tokenString) // returns parsed ID token on success. :contentReference[oaicite:5]{index=5}
	if err != nil {
		return nil, err
	}

	// Extract standard + custom claims into our struct
	var out Claims
	// Map standard OIDC fields to jwt.RegisteredClaims-like fields
	type std struct {
		Sub   string   `json:"sub"`
		Aud   []string `json:"aud"`
		Exp   int64    `json:"exp"`
		Iat   int64    `json:"iat"`
		Nbf   int64    `json:"nbf"`
		Iss   string   `json:"iss"`
		Scope string   `json:"scope,omitempty"` // space-delimited; many IdPs use this
	}
	var s std
	if err := idt.Claims(&s); err != nil {
		return nil, err
	}
	// Copy standard → RegisteredClaims
	out.RegisteredClaims.Subject = s.Sub
	out.RegisteredClaims.Issuer = s.Iss
	if len(s.Aud) > 0 {
		out.RegisteredClaims.Audience = jwt.ClaimStrings(s.Aud)
	}
	if s.Exp != 0 {
		out.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Unix(s.Exp, 0))
	}
	if s.Iat != 0 {
		out.RegisteredClaims.IssuedAt = jwt.NewNumericDate(time.Unix(s.Iat, 0))
	}
	if s.Nbf != 0 {
		out.RegisteredClaims.NotBefore = jwt.NewNumericDate(time.Unix(s.Nbf, 0))
	}

	// Extract custom fields in a second pass (tenant_id, user_id, scopes)
	var custom map[string]any
	if err := idt.Claims(&custom); err == nil {
		if v, ok := custom["tenant_id"].(string); ok {
			out.TenantID = v
		}
		if v, ok := custom["user_id"].(string); ok {
			out.UserID = v
		}
		if v, ok := custom["scope"].(string); ok && out.Scopes == "" {
			out.Scopes = v
		}
		if arr, ok := custom["scope"].([]any); ok && len(out.Scope) == 0 {
			for _, e := range arr {
				if s, ok := e.(string); ok {
					out.Scope = append(out.Scope, s)
				}
			}
		}
		if v, ok := custom["scopes"].(string); ok && out.Scopes == "" {
			out.Scopes = v
		}
	}

	// Normalize scopes to []string
	if len(out.Scope) == 0 && out.Scopes != "" {
		out.Scope = strings.Fields(out.Scopes)
	}

	// Optional authorization check (RequiredScopes)
	if len(v.opt.RequiredScopes) > 0 {
		for _, rs := range v.opt.RequiredScopes {
			if !contains(out.Scope, rs) {
				return nil, errors.New("missing required scope: " + rs)
			}
		}
	}
	return &out, nil
}

func optAudienceFirst(aud []string) string {
	if len(aud) > 0 {
		return aud[0]
	}
	return ""
}

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}
