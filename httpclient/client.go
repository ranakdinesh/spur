package httpclient

import (
	"net"
	"net/http"
	"time"

	"github.com/ranakdinesh/spur/auth/authclient"
)

func New(opt Options) *http.Client {
	// defaults
	if opt.Timeout == 0 {
		opt.Timeout = 5 * time.Second
	}
	if opt.MaxIdleConns == 0 {
		opt.MaxIdleConns = 64
	}
	if opt.MaxIdleConnsPerHost == 0 {
		opt.MaxIdleConnsPerHost = 16
	}
	if opt.IdleConnTimeout == 0 {
		opt.IdleConnTimeout = 60 * time.Second
	}
	if opt.Retries == 0 {
		opt.Retries = 2
	}
	if opt.BackoffMin == 0 {
		opt.BackoffMin = 100 * time.Millisecond
	}
	if opt.BackoffMax == 0 {
		opt.BackoffMax = 1500 * time.Millisecond
	}
	if opt.RetryOnStatuses == nil || len(opt.RetryOnStatuses) == 0 {
		opt.RetryOnStatuses = []int{502, 503, 504, 408, 425}
	}
	if !opt.IdempotentOnly {
		// leave as configured; default is true below if zero-value
	} else {
		opt.IdempotentOnly = true
	}
	if !opt.RetryOnNetwork {
		opt.RetryOnNetwork = true
	}
	if !opt.RespectRetryAfter {
		opt.RespectRetryAfter = true
	}

	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		MaxIdleConns:          opt.MaxIdleConns,
		MaxIdleConnsPerHost:   opt.MaxIdleConnsPerHost,
		IdleConnTimeout:       opt.IdleConnTimeout,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	// Wrap propagation first (adds Authorization + X-Request-Id)
	prop := authclient.AuthTransport{Base: base}

	// Wrap with retry logic
	rt := RetryTransport{
		Base:              prop,
		Retries:           opt.Retries,
		BackoffMin:        opt.BackoffMin,
		BackoffMax:        opt.BackoffMax,
		IdempotentOnly:    opt.IdempotentOnly,
		RetryOnStatuses:   opt.RetryOnStatuses,
		RetryOnNetwork:    opt.RetryOnNetwork,
		RespectRetryAfter: opt.RespectRetryAfter,
	}

	return &http.Client{
		Transport: rt,
		Timeout:   opt.Timeout,
	}
}
