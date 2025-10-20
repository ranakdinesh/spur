package httpclient

import "time"

type Options struct {
	// HTTP client-level timeout (whole request incl. body read)
	Timeout time.Duration // default 5s

	// Transport (connection) tuning
	MaxIdleConns        int           // default 64
	MaxIdleConnsPerHost int           // default 16
	IdleConnTimeout     time.Duration // default 60s

	// Retry behavior
	Retries           int           // default 2 (→ 3 total attempts)
	BackoffMin        time.Duration // default 100ms
	BackoffMax        time.Duration // default 1.5s
	IdempotentOnly    bool          // default true (retry only GET/HEAD/OPTIONS)
	RetryOnStatuses   []int         // default: 502, 503, 504, 408, 425
	RetryOnNetwork    bool          // default true (temporary net errors)
	RespectRetryAfter bool          // default true (uses server Retry-After header)
}
