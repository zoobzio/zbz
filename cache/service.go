package cache

import (
	"context"
	"fmt"
	"time"

	"zbz/cereal"
	"zbz/zlog"
)

// zCache is the singleton service layer that orchestrates cache operations
// Like zlog's service layer, this manages provider + cereal serialization
type zCache struct {
	provider       CacheProvider     // Backend provider wrapper
	serializer     cereal.CerealProvider // Cereal handles ALL serialization
	config         CacheConfig       // Service configuration
	contractName   string            // Name of the contract that created this singleton
}

// CacheConfig defines provider-agnostic cache configuration
type CacheConfig struct {
	// Service configuration
	DefaultTTL    time.Duration `json:"default_ttl"`
	KeyPrefix     string        `json:"key_prefix"`
	Serialization string        `json:"serialization"`
	
	// Connection settings (network-based providers)
	URL      string `json:"url,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database int    `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	
	// Performance settings
	PoolSize     int           `json:"pool_size,omitempty"`
	MaxRetries   int           `json:"max_retries,omitempty"`
	Timeout      time.Duration `json:"timeout,omitempty"`
	ReadTimeout  time.Duration `json:"read_timeout,omitempty"`
	WriteTimeout time.Duration `json:"write_timeout,omitempty"`
	
	// Feature flags
	EnablePipelining bool `json:"enable_pipelining,omitempty"`
	EnableCluster    bool `json:"enable_cluster,omitempty"`
	EnableSSL        bool `json:"enable_ssl,omitempty"`
	
	// Storage settings (memory/filesystem providers)
	MaxSize         int64         `json:"max_size,omitempty"`
	CleanupInterval time.Duration `json:"cleanup_interval,omitempty"`
	BaseDir         string        `json:"base_dir,omitempty"`
	Permissions     int           `json:"permissions,omitempty"`
	
	// Cluster settings
	ClusterNodes []string `json:"cluster_nodes,omitempty"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL:       1 * time.Hour,
		KeyPrefix:        "",
		Serialization:    "json",
		PoolSize:         10,
		MaxRetries:       3,
		Timeout:          5 * time.Second,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     3 * time.Second,
		MaxSize:          100 * 1024 * 1024, // 100MB
		CleanupInterval:  2 * time.Minute,
		Permissions:      0644,
	}
}

// Singleton service instance (like zlog)
var cache *zCache

// configureFromContract initializes the singleton from a contract's registration
func configureFromContract(contractName string, provider CacheProvider, config CacheConfig) error {
	// Check if we need to replace existing singleton
	if cache != nil && cache.contractName != contractName {
		zlog.Info("Replacing cache singleton",
			zlog.String("old_contract", cache.contractName),
			zlog.String("new_contract", contractName))
		
		// Close old provider
		if err := cache.provider.Close(); err != nil {
			zlog.Warn("Failed to close old provider", zlog.Err(err))
		}
	} else if cache != nil && cache.contractName == contractName {
		// Same contract, no need to replace
		return nil
	}

	// Set up cereal serialization based on config
	cerealConfig := cereal.DefaultConfig()
	cerealConfig.Name = "cache-serializer"
	cerealConfig.DefaultFormat = config.Serialization
	cerealConfig.EnableCaching = true // Enable caching for performance
	cerealConfig.EnableScoping = false // Cache typically doesn't need scoping
	
	// Create appropriate cereal provider based on serialization format
	var cerealProvider cereal.CerealProvider
	switch config.Serialization {
	case "raw":
		contract := cereal.NewRawProvider(cerealConfig)
		cerealProvider = contract.Provider()
	case "string":
		contract := cereal.NewStringProvider(cerealConfig)
		cerealProvider = contract.Provider()
	case "json":
		fallthrough
	default:
		contract := cereal.NewJSONProvider(cerealConfig)
		cerealProvider = contract.Provider()
	}

	// Create service singleton
	cache = &zCache{
		provider:     provider,
		serializer:   cerealProvider, // Cereal handles all serialization
		config:       config,
		contractName: contractName,
	}

	zlog.Info("Cache service configured from contract",
		zlog.String("contract", contractName),
		zlog.String("provider", provider.GetProvider()),
		zlog.String("serialization", config.Serialization),
		zlog.Duration("default_ttl", config.DefaultTTL))

	return nil
}



// Table creates a typed table contract from the singleton service
// This is like cache.Table[User]("users") - main usage pattern
func (c *zCache) Table[T any](name string) *TableContract[T] {
	if c == nil {
		panic("cache not configured - call cache.Configure() first")
	}

	// Cereal handles type-appropriate serialization automatically
	// No need to select serializer - cereal determines format based on data type
	return &TableContract[T]{
		name:   name,
		prefix: c.config.KeyPrefix + name + ":",
		cache:  c, // Reference back to service singleton (includes cereal serializer)
	}
}


// Provider returns the standardized provider interface
func (c *zCache) Provider() CacheProvider {
	if c == nil {
		return nil
	}
	return c.provider
}

// Config returns the current cache configuration
func (c *zCache) Config() CacheConfig {
	if c == nil {
		return CacheConfig{}
	}
	return c.config
}


// Health check operations
func (c *zCache) Ping(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("cache not configured")
	}
	return c.provider.Ping(ctx)
}

func (c *zCache) Stats(ctx context.Context) (CacheStats, error) {
	if c == nil {
		return CacheStats{}, fmt.Errorf("cache not configured")
	}
	return c.provider.Stats(ctx)
}

// Close shuts down the cache service
func (c *zCache) Close() error {
	if c == nil {
		return nil
	}
	
	zlog.Info("Closing cache service")
	err := c.provider.Close()
	cache = nil // Clear singleton
	return err
}