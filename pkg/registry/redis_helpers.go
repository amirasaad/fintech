package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisCache creates a new Redis cache for the registry
func NewRedisCache(client *redis.Client, prefix string, ttl time.Duration) *RedisCache {
	if prefix == "" {
		prefix = "registry:"
	}
	return &RedisCache{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// RedisCache implements the Cache interface using Redis
// Note: This is a simplified version. The full implementation should be in registry_redis_cache.go
type RedisCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// Get retrieves an entity from Redis cache
func (c *RedisCache) Get(ctx context.Context, id string) (Entity, bool) {
	val, err := c.client.Get(ctx, c.prefix+id).Result()
	if err != nil {
		return nil, false
	}

	var entity BaseEntity
	if err := json.Unmarshal([]byte(val), &entity); err != nil {
		return nil, false
	}

	return &entity, true
}

// Set stores an entity in Redis cache
func (c *RedisCache) Set(ctx context.Context, entity Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	ttl := c.ttl
	if ttl <= 0 {
		ttl = 24 * time.Hour // Default TTL
	}

	return c.client.Set(ctx, c.prefix+entity.ID(), data, ttl).Err()
}

// Delete removes an entity from Redis cache
func (c *RedisCache) Delete(ctx context.Context, id string) error {
	return c.client.Del(ctx, c.prefix+id).Err()
}

// Clear removes all cached entities with the prefix
func (c *RedisCache) Clear(ctx context.Context) error {
	// Note: In production, consider using SCAN with MATCH for large datasets
	iter := c.client.Scan(ctx, 0, c.prefix+"*", 0).Iterator()
	var err error

	for iter.Next(ctx) {
		if delErr := c.client.Del(ctx, iter.Val()).Err(); delErr != nil {
			return delErr
		}
	}

	if err = iter.Err(); err != nil {
		return err
	}

	return err
}

// Size returns the number of cache entries (approximate)
func (c *RedisCache) Size() int {
	// This is an approximation as SCARD is not used
	keys, err := c.client.Keys(context.Background(), c.prefix+"*").Result()
	if err != nil {
		return 0
	}
	return len(keys)
}

// NewRedisClient creates a new Redis client from a URL
func NewRedisClient(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}
