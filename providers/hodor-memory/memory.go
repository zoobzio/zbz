package memory

import (
	"fmt"
	"strings"
	"sync"
	"time"
	
	"zbz/hodor"
)

// memoryItem represents a stored item with TTL
type memoryItem struct {
	data      []byte
	expiresAt time.Time
}

// memoryStorage implements HodorProvider interface using in-memory storage
type memoryStorage struct {
	items map[string]*memoryItem
	mu    sync.RWMutex
}

// NewMemoryStorage creates a memory storage contract with type-safe native client access
// Returns a contract that can be registered as the global singleton or used independently
// Example:
//   contract := hodormemory.NewMemoryStorage(config)
//   contract.Register()  // Register as global singleton
//   memStorage := contract.Native()  // Get *memoryStorage without casting
func NewMemoryStorage(config hodor.HodorConfig) (*hodor.HodorContract[*memoryStorage], error) {
	provider := &memoryStorage{
		items: make(map[string]*memoryItem),
	}
	
	// Create and return contract
	return hodor.NewContract("memory", provider, provider, config), nil
}

// Get retrieves data by key
func (m *memoryStorage) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	item, exists := m.items[key]
	if !exists {
		return nil, fmt.Errorf("key '%s' not found", key)
	}
	
	// Check if item has expired
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		// Item expired, remove it
		delete(m.items, key)
		return nil, fmt.Errorf("key '%s' not found", key)
	}
	
	return item.data, nil
}

// Set stores data with optional TTL
func (m *memoryStorage) Set(key string, data []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	item := &memoryItem{
		data: make([]byte, len(data)),
	}
	copy(item.data, data)
	
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}
	
	m.items[key] = item
	return nil
}

// Delete removes a key
func (m *memoryStorage) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.items, key)
	return nil
}

// Exists checks if a key exists and is not expired
func (m *memoryStorage) Exists(key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	item, exists := m.items[key]
	if !exists {
		return false, nil
	}
	
	// Check if item has expired
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		// Item expired, remove it
		delete(m.items, key)
		return false, nil
	}
	
	return true, nil
}

// List returns all keys with the given prefix
func (m *memoryStorage) List(prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var keys []string
	now := time.Now()
	
	for key, item := range m.items {
		// Skip expired items
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			continue
		}
		
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	
	return keys, nil
}

// memoryDriver implements StorageDriver for memory storage
type memoryDriver struct{}

func (d *memoryDriver) DriverName() string {
	return "memory"
}

func (d *memoryDriver) Connect(config map[string]any) (storage.StorageService, error) {
	return NewMemoryProvider(), nil
}

// Auto-register this driver when imported
func init() {
	storage.RegisterDriver("memory", &memoryDriver{})
}