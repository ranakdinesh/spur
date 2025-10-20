package healthx

import (
	"context"
	"time"
)

type CheckResult struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"` // "ok" or "fail"
	LatencyMS float64 `json:"latency_ms"`
	Error     string  `json:"error,omitempty"`
}

type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

type Aggregator struct {
	checks []Checker
}

func New() *Aggregator { return &Aggregator{} }

func (a *Aggregator) Register(cs ...Checker) { a.checks = append(a.checks, cs...) }

// Results returns all check results and overall ok bool.
// If no checks are registered, readiness = true by default.
func (a *Aggregator) Results(ctx context.Context) ([]CheckResult, bool) {
	if len(a.checks) == 0 {
		return nil, true
	}
	out := make([]CheckResult, 0, len(a.checks))
	ok := true
	for _, c := range a.checks {
		start := time.Now()
		err := c.Check(ctx)
		cr := CheckResult{
			Name:      c.Name(),
			LatencyMS: float64(time.Since(start).Microseconds()) / 1e3,
		}
		if err != nil {
			cr.Status = "fail"
			cr.Error = err.Error()
			ok = false
		} else {
			cr.Status = "ok"
		}
		out = append(out, cr)
	}
	return out, ok
}
