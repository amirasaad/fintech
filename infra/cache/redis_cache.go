package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/redis/go-redis/v9"
)

// RedisExchangeRateCache implements ExchangeRateCache using Redis.
type RedisExchangeRateCache struct {
	client *redis.Client
	prefix string
	logger *slog.Logger
}

// NewRedisExchangeRateCache creates a new RedisExchangeRateCache.
func NewRedisExchangeRateCache(
	addr, password string,
	db int,
	prefix string,
) *RedisExchangeRateCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisExchangeRateCache{client: client, prefix: prefix}
}

// NewRedisExchangeRateCacheWithOptions creates a new RedisExchangeRateCache
// from redis.Options.
func NewRedisExchangeRateCacheWithOptions(
	opt *redis.Options,
	prefix string,
	logger *slog.Logger,
) *RedisExchangeRateCache {
	client := redis.NewClient(opt)
	return &RedisExchangeRateCache{client: client, prefix: prefix, logger: logger}
}

func (r *RedisExchangeRateCache) key(key string) string {
	return r.prefix + key
}

func (r *RedisExchangeRateCache) Get(key string) (*domain.ExchangeRate, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, r.key(key)).Result()
	if errors.Is(err, redis.Nil) {
		r.logger.Debug("Redis cache miss", "key", key)
		return nil, nil // cache miss
	}
	if err != nil {
		r.logger.Error("Redis cache get error", "key", key, "error", err)
		return nil, err
	}
	var rate domain.ExchangeRate
	if err := json.Unmarshal([]byte(val), &rate); err != nil {
		r.logger.Error("Redis cache unmarshal error", "key", key, "error", err)
		return nil, err
	}
	r.logger.Debug("Redis cache hit", "key", key, "rate", rate.Rate)
	return &rate, nil
}

func (r *RedisExchangeRateCache) Set(
	key string,
	rate *domain.ExchangeRate,
	ttl time.Duration,
) error {
	ctx := context.Background()
	data, err := json.Marshal(rate)
	if err != nil {
		r.logger.Error("Redis cache marshal error", "key", key, "error", err)
		return err
	}
	err = r.client.Set(ctx, r.key(key), data, ttl).Err()
	if err != nil {
		r.logger.Error("Redis cache set error", "key", key, "error", err)
		return err
	}
	r.logger.Debug("Redis cache set", "key", key, "rate", rate.Rate, "ttl", ttl)
	return nil
}

func (r *RedisExchangeRateCache) Delete(key string) error {
	ctx := context.Background()
	err := r.client.Del(ctx, r.key(key)).Err()
	if err != nil {
		r.logger.Error("Redis cache delete error", "key", key, "error", err)
		return err
	}
	r.logger.Debug("Redis cache delete", "key", key)
	return nil
}

func (r *RedisExchangeRateCache) GetLastUpdate(key string) (time.Time, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, r.key("last_update:"+key)).Result()
	if errors.Is(err, redis.Nil) {
		return time.Time{}, nil // not set
	}
	if err != nil {
		r.logger.Error(
			"Redis cache get last update error",
			"key", key,
			"error", err,
		)
		return time.Time{}, err
	}
	ts, err := time.Parse(time.RFC3339Nano, val)
	if err != nil {
		r.logger.Error(
			"Redis cache parse last update error",
			"key", key,
			"error", err,
		)
		return time.Time{}, err
	}
	return ts, nil
}

func (r *RedisExchangeRateCache) SetLastUpdate(key string, t time.Time) error {
	ctx := context.Background()
	err := r.client.Set(
		ctx,
		r.key("last_update:"+key),
		t.Format(time.RFC3339Nano),
		0,
	).Err()
	if err != nil {
		r.logger.Error(
			"Redis cache set last update error",
			"key", key,
			"error", err,
		)
		return err
	}
	r.logger.Debug("Redis cache set last update", "key", key, "timestamp", t)
	return nil
}
