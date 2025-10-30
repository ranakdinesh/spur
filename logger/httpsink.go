package logger

import (
	"bytes"
	"net/http"
	"time"
)

type HTTPSinkConfig struct {
	URL     string
	APIKey  string
	Timeout time.Duration
	Buffer  int // number of log lines buffered (dropping when full)
}

// HTTPSink implements io.Writer. It POSTs each JSON log line to the URL.
// It is non-blocking: if buffer is full, it drops lines (never stalls the app).
type HTTPSink struct {
	cfg    HTTPSinkConfig
	client *http.Client
	ch     chan []byte
}

func NewHTTPSink(cfg HTTPSinkConfig) *HTTPSink {
	s := &HTTPSink{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		ch: make(chan []byte, cfg.Buffer),
	}
	go s.loop()
	return s
}

func (s *HTTPSink) Write(p []byte) (int, error) {
	// copy to avoid reuse of underlying slice by zerolog
	cp := make([]byte, len(p))
	copy(cp, p)

	select {
	case s.ch <- cp:
	default:
		// buffer full: drop silently (best effort)
	}
	return len(p), nil
}

func (s *HTTPSink) loop() {
	for line := range s.ch {
		req, _ := http.NewRequest("POST", s.cfg.URL, bytes.NewReader(line))
		req.Header.Set("Content-Type", "application/json")
		if s.cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
		}
		// best effort: one retry after a short backoff
		if _, err := s.client.Do(req); err != nil {
			time.Sleep(100 * time.Millisecond)
			s.client.Do(req) // ignore error on retry
		}
	}
}
