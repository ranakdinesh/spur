package httpclient

import (
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"

	"strings"
	"time"
)

type RetryTransport struct {
	Base http.RoundTripper

	Retries           int
	BackoffMin        time.Duration
	BackoffMax        time.Duration
	IdempotentOnly    bool
	RetryOnStatuses   []int
	RetryOnNetwork    bool
	RespectRetryAfter bool
}

func (t RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	// Buffer body if present and small enough; otherwise, we won't retry non-idempotent anyway.
	var bodyBytes []byte
	if req.Body != nil && req.GetBody == nil {
		// Best-effort: read into memory up to a cap; large streams aren't retried.
		const capBytes = 256 << 10 // 256KB
		b, err := io.ReadAll(io.LimitReader(req.Body, capBytes))
		if err == nil {
			bodyBytes = b
			req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			// For retries we’ll reset by re-wrapping from bodyBytes.
			req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(string(bodyBytes))), nil }
		}
	}

	attempts := t.Retries + 1
	var resp *http.Response
	var err error

	for i := 0; i < attempts; i++ {
		// Clone request for safety on retries
		r := req
		if i > 0 && req.GetBody != nil {
			rc, _ := req.GetBody()
			r = req.Clone(req.Context())
			r.Body = rc
		}

		resp, err = base.RoundTrip(r)

		// Decide if we should retry
		should, wait := t.shouldRetry(r, resp, err, i)
		if !should {
			break
		}

		// Drain and close resp body before retrying to reuse connections
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		select {
		case <-time.After(wait):
		case <-req.Context().Done():
			// Abort on context cancellation
			return nil, req.Context().Err()
		}
	}
	return resp, err
}

var idempotent = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

func (t RetryTransport) shouldRetry(req *http.Request, resp *http.Response, err error, attempt int) (bool, time.Duration) {
	// No retries left
	if attempt >= t.Retries {
		return false, 0
	}

	// Only retry idempotent methods by default
	if t.IdempotentOnly && !idempotent[req.Method] {
		return false, 0
	}

	// Network errors
	if err != nil {
		if t.RetryOnNetwork && isTemporary(err) {
			return true, t.backoff(attempt)
		}
		return false, 0
	}

	// Retry-After (seconds or HTTP-date)
	if t.RespectRetryAfter && resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" && shouldOnStatus(resp.StatusCode, t.RetryOnStatuses) {
			if dur, ok := parseRetryAfter(ra); ok {
				return true, dur
			}
		}
	}

	// Retry on chosen statuses
	if resp != nil && shouldOnStatus(resp.StatusCode, t.RetryOnStatuses) {
		return true, t.backoff(attempt)
	}

	return false, 0
}

func (t RetryTransport) backoff(attempt int) time.Duration {
	// Exponential backoff with jitter between [min, max]
	// base = min * 2^attempt (capped at max), then add 0–min jitter
	base := float64(t.BackoffMin) * math.Pow(2, float64(attempt))
	if base > float64(t.BackoffMax) {
		base = float64(t.BackoffMax)
	}
	jitter := rand.Int63n(int64(t.BackoffMin))
	return time.Duration(base) + time.Duration(jitter)
}

func isTemporary(err error) bool {
	// net.Error with Temporary() or Timeout()
	var ne net.Error
	if ok := errorAs(err, &ne); ok {
		return ne.Timeout() || ne.Temporary()
	}
	// connection resets etc.
	if strings.Contains(err.Error(), "connection reset by peer") {
		return true
	}
	return false
}

// small local "errors.As" without importing errors for older Go?
func errorAs(err error, target interface{}) bool {
	switch t := target.(type) {
	case *net.Error:
		var ne net.Error
		if ok := As(err, &ne); ok {
			*t = ne
			return true
		}
	}
	return false
}

// As mirrors errors.As but keeps imports lean; if you prefer, replace with errors.As directly.
func As(err error, target interface{}) bool {
	// Just use stdlib:
	return errorsAs(err, target)
}

var errorsAs = func(err error, target interface{}) bool {
	// delegate to stdlib; injected for testability
	return asStd(err, target)
}

func asStd(err error, target interface{}) bool {
	return AsStd(err, target)
}
