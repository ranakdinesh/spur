
package renderer

import (
	"sync"
	"time"
)

type fullPageEntry struct {
	html  []byte
	etag  string
	expAt time.Time
}

type fullPageCache struct {
	mu    sync.RWMutex
	data  map[string]*fullPageEntry
	order []string
	max   int
	ttl   time.Duration
}

func newFullPageCache(max int, ttl time.Duration) *fullPageCache {
	if max <= 0 { max = 256 }
	return &fullPageCache{
		data:  make(map[string]*fullPageEntry, max),
		order: make([]string, 0, max),
		max:   max,
		ttl:   ttl,
	}
}

func (c *fullPageCache) Get(key string) (*fullPageEntry, bool) {
	now := time.Now()
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || e.expAt.Before(now) {
		if ok {
			c.mu.Lock()
			delete(c.data, key)
			c.dropFromOrder(key)
			c.mu.Unlock()
		}
		return nil, false
	}
	return e, true
}

func (c *fullPageCache) Put(key string, html []byte, etag string) {
	now := time.Now()
	c.mu.Lock(); defer c.mu.Unlock()

	if len(c.data) >= c.max {
		if len(c.order) > 0 {
			old := c.order[0]
			delete(c.data, old)
			c.order = c.order[1:]
		}
	}
	c.data[key] = &fullPageEntry{html: html, etag: etag, expAt: now.Add(c.ttl)}
	c.order = append(c.order, key)
}

func (c *fullPageCache) InvalidatePrefix(prefix string) {
	c.mu.Lock(); defer c.mu.Unlock()
	if prefix == "" {
		c.data = make(map[string]*fullPageEntry, c.max)
		c.order = c.order[:0]
		return
	}
	for k := range c.data {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.data, k)
			c.dropFromOrder(k)
		}
	}
}

func (c *fullPageCache) dropFromOrder(key string) {
	for i, v := range c.order {
		if v == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}
