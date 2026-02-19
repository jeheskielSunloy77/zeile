package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

var ErrCacheMiss = errors.New("cache miss")

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl ...time.Duration) error
	SetJSON(ctx context.Context, key string, value any, ttl ...time.Duration) error
	GetJSON(ctx context.Context, key string, dest any) error
	Delete(ctx context.Context, keys ...string) error
}

type redisCache struct {
	client *redis.Client
	cfg    *config.CacheConfig
	logger *zerolog.Logger
}

func NewRedisCache(client *redis.Client, cfg *config.CacheConfig, logger *zerolog.Logger) *redisCache {
	return &redisCache{client: client, cfg: cfg, logger: logger}
}

func (c *redisCache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.get(ctx, key)
	c.logGet("get", key, err)
	return data, err
}

func (c *redisCache) Set(ctx context.Context, key string, value []byte, ttl ...time.Duration) error {
	err := c.set(ctx, key, value, ttl...)
	c.logWrite("set", key, err, ttl...)
	return err
}

func (c *redisCache) SetJSON(ctx context.Context, key string, value any, ttl ...time.Duration) error {
	if key == "" {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		c.logWrite("set_json", key, err, ttl...)
		return err
	}
	err = c.set(ctx, key, data, ttl...)
	c.logWrite("set_json", key, err, ttl...)
	return err
}

func (c *redisCache) Delete(ctx context.Context, keys ...string) error {
	err := c.del(ctx, keys...)
	c.logDelete(ctx, keys, err)
	return err
}

func (c *redisCache) GetJSON(ctx context.Context, key string, dest any) error {
	data, err := c.get(ctx, key)
	if err != nil {
		c.logGet("get_json", key, err)
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		_ = c.del(ctx, key)
		c.logGet("get_json", key, ErrCacheMiss)
		return ErrCacheMiss
	}

	c.logGet("get_json", key, nil)
	return nil
}

func (c *redisCache) get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	return data, err
}

func (c *redisCache) set(ctx context.Context, key string, value []byte, ttl ...time.Duration) error {
	var expiration time.Duration
	if len(ttl) > 0 {
		expiration = ttl[0]
	} else {
		expiration = c.cfg.TTL
	}
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *redisCache) del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *redisCache) logGet(op, key string, err error) {

	switch {
	case err == nil:
		c.logger.Debug().
			Str("cache", "hit").
			Str("op", op).
			Str("key", key).
			Msg("cache hit")
	case errors.Is(err, ErrCacheMiss):
		c.logger.Debug().
			Str("cache", "miss").
			Str("op", op).
			Str("key", key).
			Msg("cache miss")
	default:
		c.logger.Error().
			Err(err).
			Str("cache", "error").
			Str("op", op).
			Str("key", key).
			Msg("cache get failed")
	}
}

func (c *redisCache) logWrite(op, key string, err error, ttl ...time.Duration) {

	if err != nil {
		c.logger.Error().
			Err(err).
			Str("cache", "error").
			Str("op", op).
			Str("key", key).
			Msg("cache write failed")
		return
	}

	event := c.logger.Debug().
		Str("cache", op).
		Str("key", key)
	if len(ttl) > 0 {
		event = event.Dur("ttl", ttl[0])
	}
	event.Msg("cache write")
}

func (c *redisCache) logDelete(ctx context.Context, keys []string, err error) {

	if err != nil {
		c.logger.Error().
			Err(err).
			Str("cache", "error").
			Str("op", "delete").
			Int("keys", len(keys)).
			Msg("cache delete failed")
		return
	}

	event := c.logger.Debug().Str("cache", "delete")
	if len(keys) == 1 {
		event = event.Str("key", keys[0])
	} else {
		event = event.Int("keys", len(keys))
	}
	event.Msg("cache delete")
}
