package rediskit

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewClient(ctx context.Context, opt Options) (*redis.Client, error) {
	if opt.Addr == "" {
		return nil, errors.New("rediskit: Addr is required")
	}
	if opt.PoolSize == 0 {
		opt.PoolSize = 16
	}
	if opt.DialTimeout == 0 {
		opt.DialTimeout = 2 * time.Second
	}
	if opt.ReadTimeout == 0 {
		opt.ReadTimeout = 1 * time.Second
	}
	if opt.WriteTimeout == 0 {
		opt.WriteTimeout = 1 * time.Second
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:         opt.Addr,
		Username:     opt.Username,
		Password:     opt.Password,
		DB:           opt.DB,
		PoolSize:     opt.PoolSize,
		DialTimeout:  opt.DialTimeout,
		ReadTimeout:  opt.ReadTimeout,
		WriteTimeout: opt.WriteTimeout,
	})
	// quick ping
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}
