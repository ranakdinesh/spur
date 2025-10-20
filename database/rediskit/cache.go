package rediskit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// SetJSON marshals v and stores at key with TTL.
func SetJSON(ctx context.Context, rdb *redis.Client, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, b, ttl).Err()
}

// GetJSON fetches key and unmarshals into dest. Returns (false,nil) if not found.
func GetJSON(ctx context.Context, rdb *redis.Client, key string, dest any) (bool, error) {
	s, err := rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(s, dest); err != nil {
		return false, err
	}
	return true, nil
}

func Del(ctx context.Context, rdb *redis.Client, key string) error {
	return rdb.Del(ctx, key).Err()
}
