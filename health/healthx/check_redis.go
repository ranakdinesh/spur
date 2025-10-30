package healthx

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

type redisCheck struct{ r *redis.Client }

func (redisCheck) Name() string { return "redis" }

func (c redisCheck) Check(ctx context.Context) error {
	if c.r == nil {
		return errors.New("nil client")
	}
	return c.r.Ping(ctx).Err()
}

func Redis(r *redis.Client) Checker { return redisCheck{r: r} }
