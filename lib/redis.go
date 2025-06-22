package zbz

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NewRedisClient creates a new Redis client for caching
func NewRedisClient() *redis.Client {
	redisURL := config.RedisURL()
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		Log.Fatal("Failed to parse Redis URL", zap.Error(err))
	}

	client := redis.NewClient(opt)
	
	Log.Info("Redis client initialized", zap.String("addr", opt.Addr))
	return client
}