package cache

import (
	"context"
	"fmt"
	"time"
)

// TableContract represents a typed table within the cache service
// Like cache.Table[User]("users") - provides type-safe operations
type TableContract[T any] struct {
	name       string           // Table name: "users", "sessions", etc.
	prefix     string           // Full key prefix: "myapp:users:"
	cache      *zCache          // Reference to singleton service (cereal serialization handled internally)
}

// Type-safe struct operations

// Set stores a typed value in the table
func (t *TableContract[T]) Set(ctx context.Context, key string, value T, ttl ...time.Duration) error {
	// 1. Build full key with table prefix
	fullKey := t.prefix + key
	
	// 2. Serialize value using cereal (automatic type-appropriate serialization)
	data, err := cereal.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value with cereal: %w", err)
	}
	
	// 3. Apply TTL (provided or default)
	effectiveTTL := t.cache.config.DefaultTTL
	if len(ttl) > 0 {
		effectiveTTL = ttl[0]
	}
	
	// 4. Delegate to cache provider
	return t.cache.provider.Set(ctx, fullKey, data, effectiveTTL)
}

// Get retrieves a typed value from the table
func (t *TableContract[T]) Get(ctx context.Context, key string, dest *T) error {
	fullKey := t.prefix + key
	
	data, err := t.cache.provider.Get(ctx, fullKey)
	if err != nil {
		return err
	}
	
	// Deserialize using cereal (automatic type-appropriate deserialization)
	return cereal.Unmarshal(data, dest)
}

// Delete removes a key from the table
func (t *TableContract[T]) Delete(ctx context.Context, key string) error {
	fullKey := t.prefix + key
	return t.cache.provider.Delete(ctx, fullKey)
}

// Exists checks if a key exists in the table
func (t *TableContract[T]) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := t.prefix + key
	return t.cache.provider.Exists(ctx, fullKey)
}

// Batch operations

// SetMulti stores multiple typed values in the table
func (t *TableContract[T]) SetMulti(ctx context.Context, items map[string]T, ttl ...time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	
	// Apply TTL (provided or default)
	effectiveTTL := t.cache.config.DefaultTTL
	if len(ttl) > 0 {
		effectiveTTL = ttl[0]
	}
	
	// Serialize all items and build full keys
	serializedItems := make(map[string][]byte)
	for key, value := range items {
		fullKey := t.prefix + key
		
		data, err := cereal.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to serialize value for key %s: %w", key, err)
		}
		
		serializedItems[fullKey] = data
	}
	
	return t.cache.provider.SetMulti(ctx, serializedItems, effectiveTTL)
}

// GetMulti retrieves multiple typed values from the table
func (t *TableContract[T]) GetMulti(ctx context.Context, keys []string) (map[string]T, error) {
	if len(keys) == 0 {
		return make(map[string]T), nil
	}
	
	// Build full keys
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = t.prefix + key
	}
	
	// Get data from provider
	dataMap, err := t.cache.provider.GetMulti(ctx, fullKeys)
	if err != nil {
		return nil, err
	}
	
	// Deserialize results
	result := make(map[string]T)
	for i, key := range keys {
		fullKey := fullKeys[i]
		if data, exists := dataMap[fullKey]; exists {
			var value T
			if err := cereal.Unmarshal(data, &value); err == nil {
				result[key] = value
			}
		}
	}
	
	return result, nil
}

// DeleteMulti removes multiple keys from the table
func (t *TableContract[T]) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	
	// Build full keys
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = t.prefix + key
	}
	
	return t.cache.provider.DeleteMulti(ctx, fullKeys)
}

// Advanced operations

// Keys returns keys matching a pattern within the table
func (t *TableContract[T]) Keys(ctx context.Context, pattern string) ([]string, error) {
	fullPattern := t.prefix + pattern
	keys, err := t.cache.provider.Keys(ctx, fullPattern)
	if err != nil {
		return nil, err
	}
	
	// Strip table prefix from returned keys
	prefixLen := len(t.prefix)
	for i, key := range keys {
		if len(key) > prefixLen {
			keys[i] = key[prefixLen:]
		}
	}
	
	return keys, nil
}

// TTL returns the remaining time-to-live for a key
func (t *TableContract[T]) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := t.prefix + key
	return t.cache.provider.TTL(ctx, fullKey)
}

// Expire sets the TTL for a key
func (t *TableContract[T]) Expire(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := t.prefix + key
	return t.cache.provider.Expire(ctx, fullKey, ttl)
}

// Clear removes all keys from this table (dangerous!)
func (t *TableContract[T]) Clear(ctx context.Context) error {
	// Get all keys for this table
	pattern := t.prefix + "*"
	keys, err := t.cache.provider.Keys(ctx, pattern)
	if err != nil {
		return err
	}
	
	if len(keys) == 0 {
		return nil
	}
	
	// Delete all keys for this table
	return t.cache.provider.DeleteMulti(ctx, keys)
}

// Raw operations (for special cases)

// SetRaw stores raw bytes in the table (bypasses serialization)
func (t *TableContract[T]) SetRaw(ctx context.Context, key string, value []byte, ttl ...time.Duration) error {
	fullKey := t.prefix + key
	
	effectiveTTL := t.cache.config.DefaultTTL
	if len(ttl) > 0 {
		effectiveTTL = ttl[0]
	}
	
	return t.cache.provider.Set(ctx, fullKey, value, effectiveTTL)
}

// GetRaw retrieves raw bytes from the table (bypasses deserialization)
func (t *TableContract[T]) GetRaw(ctx context.Context, key string) ([]byte, error) {
	fullKey := t.prefix + key
	return t.cache.provider.Get(ctx, fullKey)
}

// Table metadata

// Name returns the table name
func (t *TableContract[T]) Name() string {
	return t.name
}

// Prefix returns the full key prefix used by this table
func (t *TableContract[T]) Prefix() string {
	return t.prefix
}