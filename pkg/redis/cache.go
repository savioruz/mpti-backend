package redis

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/savioruz/goth/pkg/logger"
	"time"
)

//go:generate go run go.uber.org/mock/mockgen -source=cache.go -destination=mock/cache.go -package=mock github.com/savioruz/goth/pkg/redis Interface

type IRedisCache interface {
	Save(ctx context.Context, key string, value any, duration int) (err error)
	Get(ctx context.Context, key string, value any) (err error)
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context, prefix string) error
	Pipeline() IRedisCacheWithPipe
}

type iRedisCacheImpl struct {
	client *redis.Client
	log    logger.Interface
}

func NewRedisCache(client *redis.Client, log logger.Interface) IRedisCache {
	return &iRedisCacheImpl{
		client: client,
		log:    log,
	}
}

// Clear implements IRedisCache.
func (i *iRedisCacheImpl) Clear(ctx context.Context, prefix string) error {
	pipe := i.client.Pipeline()

	scan := pipe.Scan(ctx, 0, prefix, 0)
	if scan != nil {
		iter := scan.Iterator()

		for iter.Next(ctx) {
			key := iter.Val()
			pipe.Del(ctx, key)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		i.log.Error("redis - clear - failed to clear cache", err)

		return err
	}

	return nil
}

// Delete implements IRedisCache.
func (i *iRedisCacheImpl) Delete(ctx context.Context, key string) error {
	err := i.client.Del(ctx, key).Err()

	if err != nil {
		i.log.Error("redis - delete - failed to delete cache", err)

		return err
	}

	return nil
}

// Get implements IRedisCache.
func (i *iRedisCacheImpl) Get(ctx context.Context, key string, value any) (err error) {
	cacheValue, err := i.client.Get(ctx, key).Result()

	if err == nil {
		switch v := value.(type) {
		case *string:
			*v = cacheValue
		default:
			err = json.Unmarshal([]byte(cacheValue), value)

			if err != nil {
				i.log.Error("redis - get - failed to unmarshal value", err)

				return err
			}
		}
	}

	return err
}

// Pipeline implements IRedisCache.
func (i *iRedisCacheImpl) Pipeline() IRedisCacheWithPipe {
	pipe := i.client.Pipeline()

	return NewRedisCacheWithPipe(pipe, i.client, i.log)
}

// Save implements IRedisCache.
func (i *iRedisCacheImpl) Save(ctx context.Context, key string, value any, duration int) (err error) {
	var strValue []byte
	switch v := value.(type) {
	case string:
		strValue = []byte(v)
	default:
		strValue, err = json.Marshal(v)

		if err != nil {
			i.log.Error("redis - save - failed to marshal value", err)

			return err
		}
	}

	err = i.client.Set(ctx, key, strValue, time.Second*time.Duration(duration)).Err()

	if err != nil {
		i.log.Error("redis - save - failed to save value", err)

		return err
	}

	i.log.Debug("redis - save - saved value", key)

	return
}

type IRedisCacheWithPipe interface {
	Clear(ctx context.Context, prefix string) IRedisCacheWithPipe
	Delete(ctx context.Context, key string) IRedisCacheWithPipe
	Exec(ctx context.Context) error
}

type iRedisCacheWithPipeImpl struct {
	pipe   redis.Pipeliner
	client *redis.Client
	log    logger.Interface
}

func NewRedisCacheWithPipe(pipe redis.Pipeliner, redisClient *redis.Client, log logger.Interface) IRedisCacheWithPipe {
	return &iRedisCacheWithPipeImpl{
		pipe:   pipe,
		client: redisClient,
		log:    log,
	}
}

// Clear implements IRedisCacheWithPipe.
func (i *iRedisCacheWithPipeImpl) Clear(ctx context.Context, prefix string) IRedisCacheWithPipe {
	scan := i.client.Scan(ctx, 0, prefix, 0)
	iter := scan.Iterator()

	for iter.Next(ctx) {
		key := iter.Val()

		err := i.pipe.Del(ctx, key).Err()

		if err != nil {
			i.log.Error("redis - clear - failed to clear cache", err)

			return i
		}
	}

	return i
}

// Delete implements IRedisCacheWithPipe.
func (i *iRedisCacheWithPipeImpl) Delete(ctx context.Context, key string) IRedisCacheWithPipe {
	err := i.pipe.Del(ctx, key).Err()

	if err != nil {
		i.log.Error("redis - delete - failed to delete cache", err)

		return i
	}

	return i
}

// Exec implements IRedisCacheWithPipe.
func (i *iRedisCacheWithPipeImpl) Exec(ctx context.Context) error {
	_, err := i.pipe.Exec(ctx)
	if err != nil {
		i.log.Error("redis - exec - failed to exec", err)

		return err
	}

	return nil
}
