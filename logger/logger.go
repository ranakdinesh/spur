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
)

type Loggerx struct {
    Logger zerolog.Logger
    cfg    *config.Config
}

func New(cfg *config.Config) *Loggerx {
    level, err := zerolog.ParseLevel(strings.ToLower(cfg.Log.Level))
    if err != nil { level = zerolog.InfoLevel }

    var l zerolog.Logger
    if cfg.Log.Env == "development" {
        // human-friendly console writer
        cw := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
        l = zerolog.New(cw).With().Timestamp().Logger().Level(level)
    } else {
        l = log.Logger.Level(level).With().Timestamp().Logger()
    }

    return &Loggerx{Logger: l, cfg: cfg}
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
