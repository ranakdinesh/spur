package rediskit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ObtainLock tries SET NX key=value with TTL and returns a release func.
// Safe release uses a Lua compare-and-del to avoid unlocking someone else's lock.
func ObtainLock(ctx context.Context, rdb *redis.Client, key string, ttl time.Duration) (release func(context.Context) error, ok bool, err error) {
	token := randomToken()
	ok, err = rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil || !ok {
		return nil, ok, err
	}
	release = func(c context.Context) error {
		const lua = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
else
	return 0
end`
		res := rdb.Eval(c, lua, []string{key}, token)
		if res.Err() != nil {
			return res.Err()
		}
		// If 0, lock was gone or token didn't match; not an error per se.
		return nil
	}
	return release, true, nil
}

func randomToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

var ErrLockNotAcquired = errors.New("rediskit: lock not acquired")
