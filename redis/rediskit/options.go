package rediskit

import "time"

type Options struct {
	Addr     string // "localhost:6379" or "redis:6379"
	Username string
	Password string
	DB       int

	// Pool & timeouts (sane defaults applied if zero)
	PoolSize     int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}
