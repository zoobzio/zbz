package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"zbz/shared/logger"
)

// redisCache implements the Cache interface using Redis
type redisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache implementation
func NewRedisCache(redisURL string) Cache {
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Fatal("Failed to parse Redis URL", logger.Err(err))
	}

	client := redis.NewClient(opt)
	
	logger.Info("Redis cache initialized", logger.String("addr", opt.Addr))
	
	return &redisCache{
		client: client,
	}
}

// Set stores a value in Redis with expiration
func (r *redisCache) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value from Redis
func (r *redisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

// Delete removes a key from Redis
func (r *redisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in Redis
func (r *redisCache) Exists(ctx context.Context, key string) bool {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return count > 0
}

// Clear removes all keys from Redis (use with caution!)
func (r *redisCache) Clear(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}

// ContractName returns the cache implementation name
func (r *redisCache) ContractName() string {
	return "Redis Cache"
}

// ContractDescription returns the cache implementation description
func (r *redisCache) ContractDescription() string {
	return "Redis-based caching implementation for high-performance data storage"
}