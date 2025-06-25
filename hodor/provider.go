package hodor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"zbz/zlog"
)

// StorageProvider is deprecated - use HodorProvider interface from service.go
// This remains for compatibility with existing code
type StorageProvider = HodorProvider

// FileInfo provides metadata about a file in cloud storage
type FileInfo struct {
	Name    string
	Size    int64
	Mode    uint32      // File permissions
	ModTime time.Time
	IsDir   bool
}

// StorageProviderConfig provides common configuration for all providers
type StorageProviderConfig struct {
	Provider string                 `yaml:"provider"` // s3, gcs, azure, minio, etc.
	Config   map[string]interface{} `yaml:"config"`   // Provider-specific config
}

// Provider registry for pluggable storage implementations
var providers = make(map[string]ProviderFactory)
var providerMutex sync.RWMutex

// ProviderFactory creates storage providers from YAML config
type ProviderFactory func(config interface{}) (StorageProvider, error)

// RegisterProvider allows storage providers to register themselves
func RegisterProvider(name string, factory ProviderFactory) {
	providerMutex.Lock()
	defer providerMutex.Unlock()
	providers[name] = factory
	zlog.Debug("Registered storage provider", zlog.String("provider", name))
}

// GetStorageProvider creates a storage provider from provider name and config
func GetStorageProvider(providerName string, config interface{}) (StorageProvider, error) {
	providerMutex.RLock()
	factory, exists := providers[providerName]
	providerMutex.RUnlock()
	
	if !exists {
		availableProviders := ListStorageProviders()
		return nil, fmt.Errorf("unknown storage provider '%s', available: %v", providerName, availableProviders)
	}
	
	provider, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s provider: %w", providerName, err)
	}
	
	zlog.Info("Created storage provider", 
		zlog.String("provider_name", providerName),
		zlog.String("provider", provider.GetProvider()))
	
	return provider, nil
}

// ListStorageProviders returns all registered provider names
func ListStorageProviders() []string {
	providerMutex.RLock()
	defer providerMutex.RUnlock()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// MemoryProvider provides in-memory storage for testing
type MemoryProvider struct {
	data          map[string][]byte
	subscriptions map[SubscriptionID]*memorySubscription
	mu            sync.RWMutex
}

type memorySubscription struct {
	id       SubscriptionID
	key      string
	callback ChangeCallback
}

// NewMemory creates a new memory-based hodor contract
func NewMemory(config interface{}) *HodorContract {
	memProvider := &MemoryProvider{
		data:          make(map[string][]byte),
		subscriptions: make(map[SubscriptionID]*memorySubscription),
	}
	return NewContract("memory", memProvider)
}

// NewMemoryProvider creates a new memory-based storage provider (legacy)
func NewMemoryProvider(config interface{}) (HodorProvider, error) {
	return &MemoryProvider{
		data:          make(map[string][]byte),
		subscriptions: make(map[SubscriptionID]*memorySubscription),
	}, nil
}

func (m *MemoryProvider) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key '%s' not found", key)
	}
	return data, nil
}

func (m *MemoryProvider) Set(key string, data []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if this is create or update
	operation := "create"
	if _, exists := m.data[key]; exists {
		operation = "update"
	}
	
	m.data[key] = data
	// Note: TTL not implemented for memory provider
	
	// Notify subscribers
	m.notifySubscribers(key, operation)
	
	return nil
}

func (m *MemoryProvider) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if key exists before deletion
	if _, exists := m.data[key]; exists {
		delete(m.data, key)
		// Notify subscribers
		m.notifySubscribers(key, "delete")
	}
	
	return nil
}

func (m *MemoryProvider) Exists(key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.data[key]
	return exists, nil
}

func (m *MemoryProvider) List(prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var files []string
	for key := range m.data {
		if strings.HasPrefix(key, prefix) {
			files = append(files, key)
		}
	}
	return files, nil
}

func (m *MemoryProvider) Stat(key string) (FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, exists := m.data[key]
	if !exists {
		return FileInfo{}, fmt.Errorf("key '%s' not found", key)
	}
	
	return FileInfo{
		Name:    key,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
		IsDir:   false,
	}, nil
}

func (m *MemoryProvider) GetProvider() string {
	return "memory"
}

func (m *MemoryProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.data = make(map[string][]byte)
	m.subscriptions = make(map[SubscriptionID]*memorySubscription)
	return nil
}

// Subscribe to changes for a specific key
func (m *MemoryProvider) Subscribe(key string, callback ChangeCallback) (SubscriptionID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Generate unique subscription ID
	id := SubscriptionID(uuid.New().String())
	
	// Create subscription
	subscription := &memorySubscription{
		id:       id,
		key:      key,
		callback: callback,
	}
	
	m.subscriptions[id] = subscription
	
	return id, nil
}

// Unsubscribe from changes
func (m *MemoryProvider) Unsubscribe(id SubscriptionID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.subscriptions, id)
	return nil
}

// notifySubscribers notifies all subscribers about changes to a key
func (m *MemoryProvider) notifySubscribers(key, operation string) {
	// This method is called with lock already held
	event := ChangeEvent{
		Key:       key,
		Operation: operation,
		Timestamp: time.Now(),
		Size:      int64(len(m.data[key])),
	}
	
	// Notify relevant subscribers
	for _, sub := range m.subscriptions {
		if sub.key == key {
			// Call callback in goroutine to avoid blocking
			go sub.callback(event)
		}
	}
}

// Register the memory provider (legacy support)
func init() {
	RegisterProvider("memory", NewMemoryProvider)
}