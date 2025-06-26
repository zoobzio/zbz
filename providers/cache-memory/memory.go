package cachememory

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"zbz/cache"
	"zbz/zlog"
)

// memoryItem represents an item stored in memory cache
type memoryItem struct {
	value     []byte
	expiresAt time.Time
	createdAt time.Time
}

// MemoryProvider implements cache.CacheProvider using in-memory storage
type MemoryProvider struct {
	data      map[string]*memoryItem
	mutex     sync.RWMutex
	stopChan  chan bool
	startTime time.Time
	stats     struct {
		hits   int64
		misses int64
	}
	maxSize     int64 // Maximum memory usage in bytes
	currentSize int64
}

// NewMemoryCache creates a memory cache contract with type-safe native client access
// Returns a contract that can be registered as the global singleton or used independently
// Example:
//   contract := cachememory.NewMemoryCache(config)
//   contract.Register()  // Register as global singleton
//   memCache := contract.Native()  // Get *MemoryProvider without casting
func NewMemoryCache(config cache.CacheConfig) (*cache.CacheContract[*MemoryProvider], error) {
	maxSize := config.MaxSize
	if maxSize == 0 {
		maxSize = 100 * 1024 * 1024 // 100MB default
	}
	
	provider := &MemoryProvider{
		data:      make(map[string]*memoryItem),
		stopChan:  make(chan bool),
		startTime: time.Now(),
		maxSize:   maxSize,
	}
	
	// Start cleanup goroutine
	cleanupInterval := config.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Minute
	}
	go provider.cleanup(cleanupInterval)
	
	zlog.Info("Memory cache provider initialized", 
		zlog.String("max_size", fmt.Sprintf("%d bytes", maxSize)))
	
	// Create and return contract
	return cache.NewContract("memory", provider, provider, config), nil
}

// Basic operations

func (m *MemoryProvider) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	
	// Copy the value to avoid external modifications
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	
	// Check if we need to evict old item first
	if existing, exists := m.data[key]; exists {
		m.currentSize -= int64(len(existing.value))
	}
	
	// Check memory limit
	newSize := m.currentSize + int64(len(valueCopy))
	if newSize > m.maxSize {
		return fmt.Errorf("memory limit exceeded: %d bytes (max: %d)", newSize, m.maxSize)
	}
	
	m.data[key] = &memoryItem{
		value:     valueCopy,
		expiresAt: expiresAt,
		createdAt: time.Now(),
	}
	
	m.currentSize = newSize
	return nil
}

func (m *MemoryProvider) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	item, exists := m.data[key]
	if !exists {
		m.stats.misses++
		return nil, cache.ErrCacheKeyNotFound
	}
	
	// Check if expired
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		m.stats.misses++
		// Note: We don't delete here to avoid write lock in read operation
		return nil, cache.ErrCacheKeyNotFound
	}
	
	m.stats.hits++
	
	// Return a copy to avoid external modifications
	result := make([]byte, len(item.value))
	copy(result, item.value)
	
	return result, nil
}

func (m *MemoryProvider) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if item, exists := m.data[key]; exists {
		m.currentSize -= int64(len(item.value))
		delete(m.data, key)
	}
	
	return nil
}

func (m *MemoryProvider) Exists(ctx context.Context, key string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	item, exists := m.data[key]
	if !exists {
		return false, nil
	}
	
	// Check if expired
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return false, nil
	}
	
	return true, nil
}

// Batch operations

func (m *MemoryProvider) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}
	
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	result := make(map[string][]byte)
	now := time.Now()
	
	for _, key := range keys {
		if item, exists := m.data[key]; exists {
			// Check expiration
			if item.expiresAt.IsZero() || now.Before(item.expiresAt) {
				// Return copy
				value := make([]byte, len(item.value))
				copy(value, item.value)
				result[key] = value
				m.stats.hits++
			} else {
				m.stats.misses++
			}
		} else {
			m.stats.misses++
		}
	}
	
	return result, nil
}

func (m *MemoryProvider) SetMulti(ctx context.Context, items map[string]cache.CacheItem, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Calculate memory impact
	var additionalSize int64
	for key, item := range items {
		if existing, exists := m.data[key]; exists {
			additionalSize -= int64(len(existing.value))
		}
		additionalSize += int64(len(item.Value))
	}
	
	if m.currentSize+additionalSize > m.maxSize {
		return fmt.Errorf("memory limit exceeded: %d bytes (max: %d)", 
			m.currentSize+additionalSize, m.maxSize)
	}
	
	// Set all items
	for key, item := range items {
		effectiveTTL := ttl
		if item.TTL > 0 {
			effectiveTTL = item.TTL
		}
		
		expiresAt := time.Time{}
		if effectiveTTL > 0 {
			expiresAt = time.Now().Add(effectiveTTL)
		}
		
		// Copy value
		valueCopy := make([]byte, len(item.Value))
		copy(valueCopy, item.Value)
		
		// Remove old item size if exists
		if existing, exists := m.data[key]; exists {
			m.currentSize -= int64(len(existing.value))
		}
		
		m.data[key] = &memoryItem{
			value:     valueCopy,
			expiresAt: expiresAt,
			createdAt: time.Now(),
		}
		
		m.currentSize += int64(len(valueCopy))
	}
	
	return nil
}

func (m *MemoryProvider) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for _, key := range keys {
		if item, exists := m.data[key]; exists {
			m.currentSize -= int64(len(item.value))
			delete(m.data, key)
		}
	}
	
	return nil
}

// Advanced operations

func (m *MemoryProvider) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var keys []string
	now := time.Now()
	
	for key, item := range m.data {
		// Check expiration
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			continue
		}
		
		// Check pattern match
		if matched, _ := filepath.Match(pattern, key); matched {
			keys = append(keys, key)
		}
	}
	
	return keys, nil
}

func (m *MemoryProvider) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	item, exists := m.data[key]
	if !exists {
		return 0, cache.ErrCacheKeyNotFound
	}
	
	if item.expiresAt.IsZero() {
		return -1, nil // No expiration
	}
	
	now := time.Now()
	if now.After(item.expiresAt) {
		return 0, cache.ErrCacheKeyNotFound // Expired
	}
	
	return item.expiresAt.Sub(now), nil
}

func (m *MemoryProvider) Expire(ctx context.Context, key string, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	item, exists := m.data[key]
	if !exists {
		return cache.ErrCacheKeyNotFound
	}
	
	if ttl <= 0 {
		item.expiresAt = time.Time{} // No expiration
	} else {
		item.expiresAt = time.Now().Add(ttl)
	}
	
	return nil
}

// Management operations

func (m *MemoryProvider) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.data = make(map[string]*memoryItem)
	m.currentSize = 0
	
	return nil
}

func (m *MemoryProvider) Stats(ctx context.Context) (cache.CacheStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return cache.CacheStats{
		Provider:   "memory",
		Hits:       m.stats.hits,
		Misses:     m.stats.misses,
		Keys:       int64(len(m.data)),
		Memory:     m.currentSize,
		Uptime:     time.Since(m.startTime),
		LastAccess: time.Now(),
	}, nil
}

func (m *MemoryProvider) Ping(ctx context.Context) error {
	// Memory provider is always available
	return nil
}

func (m *MemoryProvider) Close() error {
	close(m.stopChan)
	return nil
}

// Provider metadata

func (m *MemoryProvider) GetProvider() string {
	return "memory"
}

func (m *MemoryProvider) GetVersion() string {
	return "1.0.0"
}

func (m *MemoryProvider) NativeClient() interface{} {
	// Return the provider itself as the native client for memory
	return m
}

// Cleanup operations

func (m *MemoryProvider) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.removeExpired()
		case <-m.stopChan:
			return
		}
	}
}

func (m *MemoryProvider) removeExpired() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	now := time.Now()
	for key, item := range m.data {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			m.currentSize -= int64(len(item.value))
			delete(m.data, key)
		}
	}
}

// Helper functions for config parsing
func getConfigInt64(config map[string]interface{}, key string, defaultValue int64) int64 {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
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

