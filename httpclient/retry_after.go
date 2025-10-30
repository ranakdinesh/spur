package httpclient

import (
	"strconv"
	"time"
)

func parseRetryAfter(v string) (time.Duration, bool) {
	// numeric seconds per RFC
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second, true
	}
	// HTTP-date format (RFC 7231); keep parser tiny: time.Parse with common layout
	// Sun, 06 Nov 1994 08:49:37 GMT
	if t, err := time.Parse(time.RFC1123, v); err == nil {
		now := time.Now().UTC()
		if t.After(now) {
			return t.Sub(now), true
		}
	}
	return 0, false
}

func shouldOnStatus(code int, list []int) bool {
	for _, c := range list {
		if code == c {
			return true
		}
	}
	return false
}
