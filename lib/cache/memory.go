package cache

import (
	"context"
	"sync"
	"time"

	"zbz/shared/logger"
)

// cacheItem represents an item stored in memory cache
type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// memoryCache implements the Cache interface using in-memory storage
type memoryCache struct {
	data     map[string]*cacheItem
	mutex    sync.RWMutex
	stopChan chan bool
}

// NewMemoryCache creates a new in-memory cache implementation
func NewMemoryCache() Cache {
	cache := &memoryCache{
		data:     make(map[string]*cacheItem),
		stopChan: make(chan bool),
	}
	
	// Start cleanup goroutine
	go cache.cleanup()
	
	logger.Info("Memory cache initialized")
	return cache
}

// Set stores a value in memory with expiration
func (m *memoryCache) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	expiresAt := time.Now().Add(expiration)
	if expiration <= 0 {
		// No expiration
		expiresAt = time.Time{}
	}
	
	// Copy the value to avoid external modifications
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	
	m.data[key] = &cacheItem{
		value:     valueCopy,
		expiresAt: expiresAt,
	}
	
	return nil
}

// Get retrieves a value from memory
func (m *memoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	item, exists := m.data[key]
	if !exists {
		return nil, ErrCacheKeyNotFound
	}
	
	// Check if expired
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		// Item expired, remove it
		delete(m.data, key)
		return nil, ErrCacheKeyNotFound
	}
	
	// Return a copy to avoid external modifications
	result := make([]byte, len(item.value))
	copy(result, item.value)
	
	return result, nil
}

// Delete removes a key from memory
func (m *memoryCache) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.data, key)
	return nil
}

// Exists checks if a key exists in memory
func (m *memoryCache) Exists(ctx context.Context, key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	item, exists := m.data[key]
	if !exists {
		return false
	}
	
	// Check if expired
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return false
	}
	
	return true
}

// Clear removes all keys from memory
func (m *memoryCache) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.data = make(map[string]*cacheItem)
	return nil
}

// ContractName returns the cache implementation name
func (m *memoryCache) ContractName() string {
	return "Memory Cache"
}

// ContractDescription returns the cache implementation description
func (m *memoryCache) ContractDescription() string {
	return "In-memory caching implementation for development and testing"
}

// cleanup removes expired items periodically
func (m *memoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
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

// removeExpired removes all expired items from the cache
func (m *memoryCache) removeExpired() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	now := time.Now()
	for key, item := range m.data {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(m.data, key)
		}
	}
}

// Stop stops the cleanup goroutine (for graceful shutdown)
func (m *memoryCache) Stop() {
	close(m.stopChan)
}