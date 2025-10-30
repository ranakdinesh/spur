package rediskit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdemOptions struct {
	TTL         time.Duration // how long to cache responses for a given key
	HeaderName  string        // default: "Idempotency-Key"
	RecordOn4xx bool          // default: false (record only 2xx/3xx)
	MaxBody     int64         // max response bytes to cache; 0 = unlimited
}

type cachedHTTP struct {
	Status int               `json:"status"`
	Header map[string]string `json:"header"`
	Body   []byte            `json:"body"`
}

// IdempotencyMiddleware implements RFC-ish behavior for unsafe methods.
// Key = hash(Method + Path + HeaderKeyValue)
func IdempotencyMiddleware(rdb *redis.Client, opt IdemOptions) func(http.Handler) http.Handler {
	if opt.TTL == 0 {
		opt.TTL = 5 * time.Minute
	}
	if opt.HeaderName == "" {
		opt.HeaderName = "Idempotency-Key"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isUnsafe(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			keyVal := strings.TrimSpace(r.Header.Get(opt.HeaderName))
			if keyVal == "" || rdb == nil {
				next.ServeHTTP(w, r)
				return
			}
			cacheKey := makeIdemKey(r.Method, r.URL.Path, keyVal)

			// Check cached response
			var cached cachedHTTP
			if ok, _ := GetJSON(r.Context(), rdb, cacheKey, &cached); ok {
				// Replay
				for k, v := range cached.Header {
					w.Header().Set(k, v)
				}
				w.WriteHeader(cached.Status)
				_, _ = w.Write(capped(cached.Body, opt.MaxBody))
				return
			}

			// Wrap writer to capture response
			rec := &recorder{ResponseWriter: w, maxBody: opt.MaxBody}
			next.ServeHTTP(rec, r)

			// Store only success by default
			if (rec.status >= 200 && rec.status < 400) || opt.RecordOn4xx {
				hmap := map[string]string{}
				for k, vv := range rec.Header() {
					if len(vv) > 0 {
						hmap[k] = vv[0]
					}
				}
				payload := cachedHTTP{
					Status: rec.status,
					Header: hmap,
					Body:   rec.buf.Bytes(),
				}
				_ = SetJSON(r.Context(), rdb, cacheKey, payload, opt.TTL)
			}
		})
	}
}

func isUnsafe(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

type recorder struct {
	http.ResponseWriter
	buf     bytes.Buffer
	maxBody int64
	status  int
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(p []byte) (int, error) {
	if r.maxBody == 0 || int64(r.buf.Len()) < r.maxBody {
		// cap writes if needed
		remain := r.maxBody - int64(r.buf.Len())
		if r.maxBody == 0 || int64(len(p)) <= remain {
			r.buf.Write(p)
		} else {
			r.buf.Write(p[:remain])
		}
	}
	return r.ResponseWriter.Write(p)
}

func makeIdemKey(method, path, keyVal string) string {
	h := sha256.New()
	io.WriteString(h, method)
	io.WriteString(h, "|")
	io.WriteString(h, path)
	io.WriteString(h, "|")
	io.WriteString(h, keyVal)
	return "idem:" + hex.EncodeToString(h.Sum(nil))
}

func capped(b []byte, max int64) []byte {
	if max == 0 || int64(len(b)) <= max {
		return b
	}
	return b[:max]
}
