package cacheredis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"zbz/cache"
	"zbz/zlog"
)

// RedisProvider implements cache.CacheProvider using Redis
type RedisProvider struct {
	client      redis.Cmdable // Supports both single instance and cluster
	config      RedisConfig
	startTime   time.Time
	isCluster   bool
}

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	URL              string        `mapstructure:"url"`
	PoolSize         int           `mapstructure:"pool_size"`
	MaxRetries       int           `mapstructure:"max_retries"`
	ReadTimeout      time.Duration `mapstructure:"read_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout"`
	EnablePipelining bool          `mapstructure:"enable_pipelining"`
	EnableCluster    bool          `mapstructure:"enable_cluster"`
	ClusterAddrs     []string      `mapstructure:"cluster_addrs"`
}

// NewProvider creates a new Redis cache provider from provider-agnostic config
// This is the provider function that users pass to cache.ConfigureWithProvider
// Example:
//   import cacheredis "zbz/providers/cache-redis"
//   cache.ConfigureWithProvider(cacheredis.NewProvider, config)
func NewProvider(config cache.CacheConfig) (cache.CacheProvider, error) {
	// Map provider-agnostic config to Redis-specific config
	cfg := RedisConfig{
		URL:              config.URL,
		PoolSize:         config.PoolSize,
		MaxRetries:       config.MaxRetries,
		ReadTimeout:      config.ReadTimeout,
		WriteTimeout:     config.WriteTimeout,
		EnablePipelining: config.EnablePipelining,
		EnableCluster:    config.EnableCluster,
		ClusterAddrs:     config.ClusterNodes,
	}
	
	// Apply defaults if not set
	if cfg.URL == "" && config.Host != "" {
		port := config.Port
		if port == 0 {
			port = 6379
		}
		cfg.URL = fmt.Sprintf("redis://%s:%d/%d", config.Host, port, config.Database)
		if config.Username != "" {
			cfg.URL = fmt.Sprintf("redis://%s:%s@%s:%d/%d", config.Username, config.Password, config.Host, port, config.Database)
		}
	} else if cfg.URL == "" {
		cfg.URL = "redis://localhost:6379"
	}
	
	var client redis.Cmdable
	var isCluster bool
	
	if cfg.EnableCluster && len(cfg.ClusterAddrs) > 0 {
		// Redis Cluster mode
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.ClusterAddrs,
			PoolSize:     cfg.PoolSize,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		})
		isCluster = true
		zlog.Info("Redis cluster cache provider initialized", 
			zlog.Strings("addrs", cfg.ClusterAddrs))
	} else {
		// Single Redis instance
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid Redis URL: %w", err)
		}
		
		// Apply configuration to options
		opt.PoolSize = cfg.PoolSize
		opt.MaxRetries = cfg.MaxRetries
		opt.ReadTimeout = cfg.ReadTimeout
		opt.WriteTimeout = cfg.WriteTimeout
		
		client = redis.NewClient(opt)
		zlog.Info("Redis cache provider initialized", 
			zlog.String("addr", opt.Addr))
	}
	
	provider := &RedisProvider{
		client:    client,
		config:    cfg,
		startTime: time.Now(),
		isCluster: isCluster,
	}
	
	// Create and return contract
	return cache.NewContract("redis", provider, client, config), nil
}

// Basic operations

func (r *RedisProvider) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisProvider) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, cache.ErrCacheKeyNotFound
		}
		return nil, err
	}
	return []byte(val), nil
}

func (r *RedisProvider) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisProvider) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Batch operations with pipeline support

func (r *RedisProvider) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}
	
	// Use pipeline for efficiency
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}
	
	result := make(map[string][]byte)
	for i, cmd := range cmds {
		if cmd.Err() == nil {
			result[keys[i]] = []byte(cmd.Val())
		}
		// Ignore redis.Nil errors (cache misses)
	}
	
	return result, nil
}

func (r *RedisProvider) SetMulti(ctx context.Context, items map[string]cache.CacheItem, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	
	pipe := r.client.Pipeline()
	for key, item := range items {
		effectiveTTL := ttl
		if item.TTL > 0 {
			effectiveTTL = item.TTL
		}
		pipe.Set(ctx, key, item.Value, effectiveTTL)
	}
	
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisProvider) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	
	// Convert to interface{} slice for Del command
	keyInterfaces := make([]interface{}, len(keys))
	for i, key := range keys {
		keyInterfaces[i] = key
	}
	
	return r.client.Del(ctx, keyInterfaces...).Err()
}

// Advanced operations

func (r *RedisProvider) Keys(ctx context.Context, pattern string) ([]string, error) {
	// Use SCAN instead of KEYS for better performance in production
	var keys []string
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}
	
	return keys, nil
}

func (r *RedisProvider) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	
	if ttl < 0 {
		if ttl == -2 {
			return 0, cache.ErrCacheKeyNotFound // Key doesn't exist
		}
		return -1, nil // No expiration set
	}
	
	return ttl, nil
}

func (r *RedisProvider) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// Management operations

func (r *RedisProvider) Clear(ctx context.Context) error {
	if r.isCluster {
		// For cluster, we need to flush each node
		return fmt.Errorf("FLUSHALL not supported in cluster mode - use Keys() and DeleteMulti() instead")
	}
	return r.client.FlushAll(ctx).Err()
}

func (r *RedisProvider) Stats(ctx context.Context) (cache.CacheStats, error) {
	info, err := r.client.Info(ctx, "stats", "memory", "clients").Result()
	if err != nil {
		return cache.CacheStats{}, err
	}
	
	// Parse Redis INFO output
	stats := cache.CacheStats{
		Provider:   "redis",
		Uptime:     time.Since(r.startTime),
		LastAccess: time.Now(),
	}
	
	// Parse key statistics from INFO output
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key, value := parts[0], parts[1]
				switch key {
				case "keyspace_hits":
					if hits, err := strconv.ParseInt(value, 10, 64); err == nil {
						stats.Hits = hits
					}
				case "keyspace_misses":
					if misses, err := strconv.ParseInt(value, 10, 64); err == nil {
						stats.Misses = misses
					}
				case "used_memory":
					if memory, err := strconv.ParseInt(value, 10, 64); err == nil {
						stats.Memory = memory
					}
				}
			}
		}
	}
	
	// Get key count from INFO keyspace
	keyspaceInfo, err := r.client.Info(ctx, "keyspace").Result()
	if err == nil {
		keyspaceLines := strings.Split(keyspaceInfo, "\r\n")
		for _, line := range keyspaceLines {
			if strings.HasPrefix(line, "db") && strings.Contains(line, "keys=") {
				parts := strings.Split(line, ",")
				for _, part := range parts {
					if strings.HasPrefix(part, "keys=") {
						keyCount := strings.TrimPrefix(part, "keys=")
						if keys, err := strconv.ParseInt(keyCount, 10, 64); err == nil {
							stats.Keys = keys
							break
						}
					}
				}
			}
		}
	}
	
	return stats, nil
}

func (r *RedisProvider) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisProvider) Close() error {
	switch client := r.client.(type) {
	case *redis.Client:
		return client.Close()
	case *redis.ClusterClient:
		return client.Close()
	default:
		return nil
	}
}

// Provider metadata

func (r *RedisProvider) GetProvider() string {
	if r.isCluster {
		return "redis-cluster"
	}
	return "redis"
}

func (r *RedisProvider) GetVersion() string {
	return "7.0.0" // Redis version - could be detected dynamically
}

func (r *RedisProvider) NativeClient() interface{} {
	return r.client
}

// Helper functions for config parsing
func getConfigString(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getConfigInt(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

func getConfigDuration(config map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case time.Duration:
			return v
		case string:
			if duration, err := time.ParseDuration(v); err == nil {
				return duration
			}
		}
	}
	return defaultValue
}

func getConfigBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func getConfigStringSlice(config map[string]interface{}, key string) []string {
	if val, ok := config[key]; ok {
		if slice, ok := val.([]string); ok {
			return slice
		}
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, v := range slice {
				if str, ok := v.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return nil
}

// Legacy provider registration for backward compatibility with old cache system
func init() {
	// Convert legacy registration to new provider function
	cache.RegisterProvider("redis", func(config map[string]interface{}) (cache.CacheProvider, error) {
		// Convert map config to structured config
		structuredConfig := cache.CacheConfig{
			URL:              getConfigString(config, "url", "redis://localhost:6379"),
			PoolSize:         getConfigInt(config, "pool_size", 10),
			MaxRetries:       getConfigInt(config, "max_retries", 3),
			ReadTimeout:      getConfigDuration(config, "read_timeout", 5*time.Second),
			WriteTimeout:     getConfigDuration(config, "write_timeout", 3*time.Second),
			EnablePipelining: getConfigBool(config, "enable_pipelining", true),
			EnableCluster:    getConfigBool(config, "enable_cluster", false),
			ClusterNodes:     getConfigStringSlice(config, "cluster_addrs"),
		}
		return NewProvider(structuredConfig)
	})
}