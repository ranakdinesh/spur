package healthx

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type response struct {
	Status    string        `json:"status"` // "ok" or "fail"
	Checks    []CheckResult `json:"checks,omitempty"`
	ElapsedMS float64       `json:"elapsed_ms"`
}

// Mount adds /health/live and /health/ready under the given router.
//
// - /health/live: always OK (process is up)
// - /health/ready: runs the registered checks; 200 if all pass, else 503
func Mount(r chi.Router, agg *Aggregator) {
	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, response{Status: "ok"})
	})

	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		results, ok := agg.Results(r.Context())
		code := http.StatusOK
		status := "ok"
		if !ok {
			code = http.StatusServiceUnavailable
			status = "fail"
		}
		writeJSON(w, code, response{
			Status:    status,
			Checks:    results,
			ElapsedMS: float64(time.Since(start).Microseconds()) / 1e3,
		})
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// Helper if you want to call readiness programmatically (e.g., during startup probes)
func Ready(ctx context.Context, agg *Aggregator) (ok bool) {
	_, ok = agg.Results(ctx)
	return ok
}
