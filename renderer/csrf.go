
package renderer

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

type csrfProvider struct{ secret []byte }

func newCSRFProvider(secret []byte) *csrfProvider {
	if len(secret) == 0 { secret = mustRandom(32) }
	return &csrfProvider{secret: secret}
}

func (c *csrfProvider) Token() string {
	nonce := mustRandom(16)
	ts := []byte(strconv.FormatInt(time.Now().Unix(), 10))
	m := hmac.New(sha256.New, c.secret)
	m.Write(nonce); m.Write([]byte{':'}); m.Write(ts)
	mac := m.Sum(nil)
	raw := append(nonce, ':'); raw = append(raw, ts...); raw = append(raw, ':'); raw = append(raw, mac...)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func mustRandom(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil { panic(err) }
	return b
}
