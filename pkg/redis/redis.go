package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *redis.Client
}

func New(addr, password string, db int) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &Redis{Client: client}, nil
}

func (r *Redis) Close() {
	if r.Client != nil {
		r.Client.Close()
	}
}

func (r *Redis) Ping(ctx context.Context) error {
	if err := r.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis: ping failed: %w", err)
	}

	return nil
}
