package logger

import (
	"context"
	"io"
	"strconv"
	"strings"

	"fmt"
	"github.com/rs/zerolog"
	"os"

	"time"
)

// ---------- Public API ----------

// Options configures the logger. Keep stdout always; optionally tee to a remote sink.
type Options struct {
	// Format/Mode
	Dev bool // pretty console when true; JSON otherwise

	// Optional remote HTTP sink (non-blocking, best-effort)
	EnableHTTPSink bool
	HTTPURL        string // e.g. http://logger-service.infra.svc:8080/api/v1/logs
	HTTPAPIKey     string
	HTTPTimeout    time.Duration // default 1s
	Buffer         int           // default 1024 log lines in memory
}

type Loggerx struct {
	l zerolog.Logger
}

// NewWithOptions is the preferred constructor.
func NewWithOptions(opts Options) *Loggerx {
	// ---------- Global zerolog config ----------
	// Level (default Info; override via LOG_LEVEL: debug|info|warn|error)
	zerolog.SetGlobalLevel(parseLevel(getEnv("LOG_LEVEL", "info")))

	// ---- Global zerolog field names and caller shortener ----
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = "ts"
	zerolog.LevelFieldName = "lvl"
	zerolog.MessageFieldName = "msg"
	zerolog.CallerFieldName = "caller"
	// Trim caller paths like "../../../../go/pkg/mod/.../file.go:123" -> "pkg/file.go:123"
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		parts := strings.Split(file, "/")
		if len(parts) > 2 {
			file = strings.Join(parts[len(parts)-2:], "/")
		}
		return fmt.Sprintf("%s:%d", file, line)
	}
	writers := make([]io.Writer, 0, 2)

	// Always keep stdout for kubectl logs / local dev.
	if opts.Dev {
		cw := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05.000",
		}
		// Short caller in console view as well
		cw.FormatCaller = func(i interface{}) string {
			if caller, ok := i.(string); ok {
				parts := strings.Split(caller, "/")
				if len(parts) > 2 {
					return parts[len(parts)-2] + "/" + parts[len(parts)-1]
				}
				return caller
			}
			return fmt.Sprintf("%v", i)
		}
		writers = append(writers, cw)

	} else {
		writers = append(writers, os.Stdout)
	}

	// Optional remote sink (HTTP).
	if opts.EnableHTTPSink && opts.HTTPURL != "" {
		hw := NewHTTPSink(HTTPSinkConfig{
			URL:     opts.HTTPURL,
			APIKey:  opts.HTTPAPIKey,
			Timeout: firstNonZero(opts.HTTPTimeout, time.Second),
			Buffer:  firstNonZeroInt(opts.Buffer, 1024),
		})
		writers = append(writers, hw) // tee: stdout + remote
	}

	mw := io.MultiWriter(writers...)

	// Caller points to the app callsite (skip wrappers).

	base := zerolog.New(mw).With().Caller().Timestamp().CallerWithSkipFrameCount(2).Logger()
	// One-time sanity probe so you SEE something if wiring is correct.
	base.Debug().Str("dev", strconv.FormatBool(opts.Dev)).Msg("logger online")
	return &Loggerx{l: base}
}

// New keeps backward compatibility with previous code paths.
func New(dev bool) *Loggerx { return NewWithOptions(Options{Dev: dev}) }

// With adds structured fields.
func (x *Loggerx) With(kv ...interface{}) *Loggerx {
	return &Loggerx{l: x.l.With().Fields(kv).Logger()}
}

// Accessors (context-aware): they attach trace/tenant/user if present in ctx.
func (x *Loggerx) Info(ctx context.Context) *zerolog.Event  { return bindCtx(x.l, ctx).Info() }
func (x *Loggerx) Error(ctx context.Context) *zerolog.Event { return bindCtx(x.l, ctx).Error() }
func (x *Loggerx) Warn(ctx context.Context) *zerolog.Event  { return bindCtx(x.l, ctx).Warn() }
func (x *Loggerx) Debug(ctx context.Context) *zerolog.Event { return bindCtx(x.l, ctx).Debug() }

// Logger returns the underlying zerolog (advanced usage).
func (x *Loggerx) Logger() zerolog.Logger { return x.l }

// ---------- Context helpers (stable API you can use anywhere) ----------

type ctxKey string

const (
	ctxKeyTraceID  ctxKey = "trace_id"
	ctxKeyTenantID ctxKey = "tenant_id"
	ctxKeyUserID   ctxKey = "user_id"
)

// WithTraceID attaches a trace ID into context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID, traceID)
}
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ctxKeyTenantID, tenantID)
}
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

func TraceIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyTraceID).(string)
	return v, ok
}
func TenantIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyTenantID).(string)
	return v, ok
}
func UserIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(string)
	return v, ok
}

// ---------- Internal ----------

func bindCtx(l zerolog.Logger, ctx context.Context) *zerolog.Logger {
	if ctx == nil {
		return &l
	}
	ev := l.With()
	if v, ok := TraceIDFrom(ctx); ok && v != "" {
		ev = ev.Str("trace_id", v)
	}
	if v, ok := TenantIDFrom(ctx); ok && v != "" {
		ev = ev.Str("tenant_id", v)
	}
	if v, ok := UserIDFrom(ctx); ok && v != "" {
		ev = ev.Str("user_id", v)
	}
	ll := ev.Logger()
	return &ll
}

func firstNonZero(v, d time.Duration) time.Duration {
	if v == 0 {
		return d
	}
	return v
}
func firstNonZeroInt(v, d int) int {
	if v == 0 {
		return d
	}
	return v
}

// ---------- helpers ----------
func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info", "":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error", "err":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	case "disabled", "off", "none":
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}
