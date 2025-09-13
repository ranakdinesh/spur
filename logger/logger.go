package logger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ranakdinesh/spur/config"

	"context"
	"errors"
	"net/url"
	"sync"
)

type Loggerx struct {
	Logger      zerolog.Logger
	cfg         *config.Config
	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
	redact      map[string]struct{}
}

func New(cfg *config.Config) *Loggerx {
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Log.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}
	var l zerolog.Logger
	if cfg.Log.Env == "development" || cfg.Log.DevMode {
		cw := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		l = zerolog.New(cw).
			With().
			Timestamp().
			Str("service", cfg.AppName). // include service always
			Caller().
			Logger().
			Level(level)
	} else {
		l = log.Logger.
			Level(level).
			With().
			Timestamp().
			Str("service", cfg.AppName).
			Logger()
	}

	// build redaction set
	redact := map[string]struct{}{}
	for _, k := range strings.Split(cfg.Log.RedactKeys, ",") {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" {
			redact[k] = struct{}{}
		}
	}

	return &Loggerx{Logger: l, cfg: cfg, redact: redact}
}

// ForwardToService sends a structured log to external logger-service (best-effort).
func (l *Loggerx) ForwardToService(event map[string]any) {
	if l.cfg.Log.Env == "development" || l.cfg.Log.LoggerServiceURL == "" {
		return
	}
	b, _ := json.Marshal(event)
	req, _ := http.NewRequest(http.MethodPost, l.cfg.Log.LoggerServiceURL, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		l.Logger.Warn().Err(err).Msg("failed to forward log")
		return
	}
	_ = resp.Body.Close()
}
func (l *Loggerx) LogKV(ctx context.Context, level zerolog.Level, msg string, fields map[string]any) {
	e := l.Logger.WithLevel(level).Str("service", l.cfg.AppName).Timestamp()

	// Context: if you propagate trace IDs, request IDs, user IDs in ctx, extract here
	if rid := ctx.Value("request_id"); rid != nil {
		e = e.Str("request_id", rid.(string))
	}
	if tid := ctx.Value("trace_id"); tid != nil {
		e = e.Str("trace_id", tid.(string))
	}
	// emit local log (dev-friendly if DevMode)
	e.Fields(fields).Msg(msg)

	// forward if prod and configured
	if !(l.cfg.Log.Env == "development" || l.cfg.Log.DevMode) && l.cfg.Log.LoggerServiceURL != "" {
		l.forward(ctx, level, msg, fields)
	}
}

// Convenience wrappers
func (l *Loggerx) Info(ctx context.Context, msg string, fields map[string]any) {
	l.LogKV(ctx, zerolog.InfoLevel, msg, fields)
}
func (l *Loggerx) Warn(ctx context.Context, msg string, fields map[string]any) {
	l.LogKV(ctx, zerolog.WarnLevel, msg, fields)
}
func (l *Loggerx) Error(ctx context.Context, msg string, fields map[string]any) {
	l.LogKV(ctx, zerolog.ErrorLevel, msg, fields)
}
func (l *Loggerx) Debug(ctx context.Context, msg string, fields map[string]any) {
	l.LogKV(ctx, zerolog.DebugLevel, msg, fields)
}

// redact sensitive info in fields (shallow)
func (l *Loggerx) sanitize(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		lk := strings.ToLower(k)
		if _, bad := l.redact[lk]; bad {
			out[k] = "***REDACTED***"
			continue
		}
		out[k] = v
	}
	return out
}

func (l *Loggerx) forward(ctx context.Context, level zerolog.Level, msg string, fields map[string]any) {
	if l.cfg.Log.LoggerServiceURL == "" {
		return
	}
	evt := map[string]any{
		"service": l.cfg.AppName,
		"level":   level.String(),
		"time":    time.Now().UTC().Format(time.RFC3339Nano),
		"message": msg,
		"fields":  l.sanitize(fields),
	}
	if rid := ctx.Value("request_id"); rid != nil {
		evt["request_id"] = rid
	}
	if tid := ctx.Value("trace_id"); tid != nil {
		evt["trace_id"] = tid
	}

	b, _ := json.Marshal(evt)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, l.cfg.Log.LoggerServiceURL, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	// attach bearer if we can obtain it
	if token, _ := l.getAccessToken(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		l.Logger.Warn().Err(err).Msg("logger forward failed")
		return
	}
	_ = resp.Body.Close()
}

// OAuth2 Client Credentials (token caching)
func (l *Loggerx) getAccessToken(ctx context.Context) (string, error) {
	l.mu.Lock()
	if time.Now().Before(l.tokenExpiry.Add(-15*time.Second)) && l.accessToken != "" {
		tok := l.accessToken
		l.mu.Unlock()
		return tok, nil
	}
	l.mu.Unlock()

	if l.cfg.Log.OAuthTokenURL == "" || l.cfg.Log.ClientID == "" || l.cfg.Log.ClientSecret == "" {
		return "", errors.New("oauth client-credentials not configured")
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", l.cfg.Log.ClientID)
	form.Set("client_secret", l.cfg.Log.ClientSecret)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, l.cfg.Log.OAuthTokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("token endpoint error: " + resp.Status)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}

	l.mu.Lock()
	l.accessToken = tok.AccessToken
	// be defensive on expiry; default to 5 minutes if missing
	if tok.ExpiresIn <= 0 {
		tok.ExpiresIn = 300
	}
	l.tokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	l.mu.Unlock()

	return tok.AccessToken, nil
}

// WithCtx creates an event pre-populated with service + context IDs.
// Usage:
//
//	l.WithCtx(ctx).Str("key","val").Msg("something")
func (l *Loggerx) WithCtx(ctx context.Context, level zerolog.Level) *zerolog.Event {
	e := l.Logger.WithLevel(level).
		Str("service", l.cfg.AppName).
		Timestamp()

	if rid := ctx.Value("request_id"); rid != nil {
		if s, ok := rid.(string); ok {
			e = e.Str("request_id", s)
		}
	}
	if tid := ctx.Value("trace_id"); tid != nil {
		if s, ok := tid.(string); ok {
			e = e.Str("trace_id", s)
		}
	}
	return e
}
