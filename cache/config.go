package cache

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// FullCacheConfig represents complete cache configuration including provider setup
type FullCacheConfig struct {
	Provider ProviderConfig `yaml:"provider" json:"provider"`
	Service  CacheConfig    `yaml:"service" json:"service"`
}

// ProviderConfig defines how to create a cache provider
type ProviderConfig struct {
	Type   string                 `yaml:"type" json:"type"`     // "redis", "memory", "filesystem"
	Config map[string]interface{} `yaml:"config" json:"config"` // Provider-specific configuration
}

// loadConfigFromYAML loads configuration from YAML file
func loadConfigFromYAML(filename string) (FullCacheConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return FullCacheConfig{}, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	var config FullCacheConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return FullCacheConfig{}, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return config, nil
}

// loadConfigFromEnv creates configuration from environment variables
func loadConfigFromEnv() (FullCacheConfig, error) {
	providerType := getEnvString("CACHE_PROVIDER", "memory")
	
	var providerConfig map[string]interface{}
	var serviceConfig CacheConfig

	switch providerType {
	case "redis":
		providerConfig = map[string]interface{}{
			"url":               getEnvString("REDIS_URL", "redis://localhost:6379"),
			"pool_size":         getEnvInt("REDIS_POOL_SIZE", 10),
			"max_retries":       getEnvInt("REDIS_MAX_RETRIES", 3),
			"read_timeout":      getEnvString("REDIS_READ_TIMEOUT", "5s"),
			"write_timeout":     getEnvString("REDIS_WRITE_TIMEOUT", "3s"),
			"enable_pipelining": getEnvBool("REDIS_ENABLE_PIPELINING", true),
		}
	case "memory":
		providerConfig = map[string]interface{}{
			"max_size":         getEnvInt64("MEMORY_CACHE_MAX_SIZE", 100*1024*1024), // 100MB
			"cleanup_interval": getEnvString("MEMORY_CACHE_CLEANUP_INTERVAL", "2m"),
		}
	case "filesystem":
		providerConfig = map[string]interface{}{
			"base_dir":    getEnvString("FILESYSTEM_CACHE_DIR", "/tmp/zbz-cache"),
			"permissions": getEnvInt("FILESYSTEM_CACHE_PERMISSIONS", 0644),
		}
	default:
		return FullCacheConfig{}, fmt.Errorf("unsupported cache provider: %s", providerType)
	}

	serviceConfig = CacheConfig{
		DefaultTTL:    getEnvDuration("CACHE_DEFAULT_TTL", 1*time.Hour),
		KeyPrefix:     getEnvString("CACHE_KEY_PREFIX", ""),
		Serialization: getEnvString("CACHE_SERIALIZATION", "json"),
	}

	return FullCacheConfig{
		Provider: ProviderConfig{
			Type:   providerType,
			Config: providerConfig,
		},
		Service: serviceConfig,
	}, nil
}

// Example YAML configuration template
func GenerateConfigTemplate() string {
	return `# ZBZ Cache Configuration
provider:
  type: "redis"  # redis, memory, filesystem
  config:
    # Redis configuration
    url: "redis://localhost:6379"
    pool_size: 10
    max_retries: 3
    read_timeout: "5s"
    write_timeout: "3s"
    enable_pipelining: true
    
    # Memory configuration (if type: memory)
    # max_size: 104857600  # 100MB
    # cleanup_interval: "2m"
    
    # Filesystem configuration (if type: filesystem)  
    # base_dir: "/tmp/zbz-cache"
    # permissions: 0644

service:
  default_ttl: "1h"
  key_prefix: "myapp:"
  serialization: "json"  # json, msgpack
`
}

// Environment variable helpers
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := parseConfigInt(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := parseConfigInt64(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// Config parsing helpers
func parseConfigInt(value string) (int, error) {
	return strconv.Atoi(value)
}

func parseConfigInt64(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}