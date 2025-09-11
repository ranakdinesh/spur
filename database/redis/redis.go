package redis

import (
    "context"

    rds "github.com/redis/go-redis/v9"

    "github.com/ranakdinesh/spur/config"
    "github.com/ranakdinesh/spur/logger"
)

type Redis struct {
    Client *rds.Client
    log    *logger.Loggerx
}

func New(ctx context.Context, cfg *config.Config, log *logger.Loggerx) (*Redis, error) {
    c := rds.NewClient(&rds.Options{
        Addr: cfg.Redis.Addr,
        Password: cfg.Redis.Password,
        DB: cfg.Redis.DB,
    })
    if err := c.Ping(ctx).Err(); err != nil {
        return nil, err
    }
    return &Redis{Client: c, log: log}, nil
}

func (r *Redis) Close() { _ = r.Client.Close() }
