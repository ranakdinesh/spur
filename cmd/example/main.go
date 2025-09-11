package main

import (
    "context"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"

    "github.com/ranakdinesh/spur/spur"
)

func main() {
    ctx := context.Background()
    s, err := spur.New(ctx, nil)
    if err != nil { panic(err) }

    // HTTP route group (parent app)
    if s.HTTP != nil {
        s.HTTP.MountGroup("/api", func(r chi.Router) {
            r.Get("/hello", func(w http.ResponseWriter, req *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.Write([]byte(`{"msg":"world"}`))
            })
        })
    }

    // start everything
    if err := s.Run(context.Background()); err != nil {
        s.Log.Logger.Error().Err(err).Msg("spur stopped with error")
    }

    // give some time to flush logs, exporters, etc.
    time.Sleep(200 * time.Millisecond)
}
