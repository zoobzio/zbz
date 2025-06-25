package hodor

import (
	"fmt"
	"sync"
	"time"

	"zbz/zlog"
)

// Types for reactive operations
type SubscriptionID string

type ChangeEvent struct {
	Key       string    `json:"key"`
	Operation string    `json:"operation"` // "create", "update", "delete"
	Timestamp time.Time `json:"timestamp"`
	ETag      string    `json:"etag,omitempty"`    // For change detection
	Size      int64     `json:"size,omitempty"`    // File size
}

type ChangeCallback func(event ChangeEvent)

// HodorProvider defines the interface that storage providers implement
// This is the standardized interface the service uses to interact with storage backends
type HodorProvider interface {
	// Core storage operations
	Get(key string) ([]byte, error)
	Set(key string, data []byte, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
	List(prefix string) ([]string, error)
	
	// Metadata operations
	Stat(key string) (FileInfo, error)
	
	// Reactive operations
	Subscribe(key string, callback ChangeCallback) (SubscriptionID, error)
	Unsubscribe(id SubscriptionID) error
	
	// Provider info
	GetProvider() string // Returns provider name for debugging
	Close() error       // Cleanup resources
}

// Private concrete hodor instance
var hodor *zHodor

// HodorService defines the interface for storage contract registry
// Service manages contract registration and discovery
type HodorService interface {
	// Register a contract (called by contracts)
	RegisterContract(alias string, provider HodorProvider) error
	
	// Unregister contract
	Unregister(alias string) error
	
	// List all registered contracts
	List() []ContractInfo
	
	// Get contract info
	Status(alias string) (ContractStatus, error)
	
	// Clean shutdown
	Close() error
}

// zHodor implements the HodorService interface for contract registry
type zHodor struct {
	contracts map[string]*registeredContract // alias → registered contract
	mu        sync.RWMutex                    // protects all state
}

// registeredContract represents a single registered storage contract
type registeredContract struct {
	alias     string
	provider  HodorProvider
	createdAt time.Time
	status    ContractStatus
}

// ContractInfo provides information about a registered contract
type ContractInfo struct {
	Alias     string    `json:"alias"`
	Provider  string    `json:"provider"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ContractStatus represents the current state of a contract
type ContractStatus struct {
	State ContractState `json:"state"`
	Error string        `json:"error,omitempty"`
}

// ContractState represents possible contract states
type ContractState string

const (
	ContractStateActive      ContractState = "active"
	ContractStateError       ContractState = "error"
	ContractStateUnregistered ContractState = "unregistered"
)


// init automatically sets up the global hodor service
func init() {
	hodor = &zHodor{
		contracts: make(map[string]*registeredContract),
	}
	zlog.Info("Initialized hodor service")
}

// RegisterContract registers a contract for service discovery
func (h *zHodor) RegisterContract(alias string, provider HodorProvider) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Check if alias already exists
	if _, exists := h.contracts[alias]; exists {
		return fmt.Errorf("contract alias '%s' already exists", alias)
	}
	
	// Create registered contract record
	contract := &registeredContract{
		alias:     alias,
		provider:  provider,
		createdAt: time.Now(),
		status: ContractStatus{
			State: ContractStateActive,
		},
	}
	
	h.contracts[alias] = contract
	
	zlog.Info("Registered contract", 
		zlog.String("alias", alias),
		zlog.String("provider", provider.GetProvider()))
	
	return nil
}

// Unregister removes a contract from the registry
func (h *zHodor) Unregister(alias string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	contract, exists := h.contracts[alias]
	if !exists {
		return fmt.Errorf("contract alias '%s' not found", alias)
	}
	
	// Close provider resources
	if err := contract.provider.Close(); err != nil {
		zlog.Warn("Error closing provider", 
			zlog.String("alias", alias),
			zlog.Err(err))
	}
	
	// Remove from registry
	delete(h.contracts, alias)
	
	zlog.Info("Unregistered contract", 
		zlog.String("alias", alias))
	
	return nil
}

// List returns information about all registered contracts
func (h *zHodor) List() []ContractInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	infos := make([]ContractInfo, 0, len(h.contracts))
	for _, contract := range h.contracts {
		infos = append(infos, ContractInfo{
			Alias:     contract.alias,
			Provider:  contract.provider.GetProvider(),
			Status:    string(contract.status.State),
			CreatedAt: contract.createdAt,
		})
	}
	
	return infos
}

// Status returns the status of a specific contract
func (h *zHodor) Status(alias string) (ContractStatus, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	contract, exists := h.contracts[alias]
	if !exists {
		return ContractStatus{}, fmt.Errorf("contract alias '%s' not found", alias)
	}
	
	return contract.status, nil
}

// Close shuts down all contracts and cleans up
func (h *zHodor) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	var errors []error
	
	// Close all providers
	for alias, contract := range h.contracts {
		if err := contract.provider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close provider %s: %w", alias, err))
		}
	}
	
	// Clear registry
	h.contracts = make(map[string]*registeredContract)
	
	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}
	
	zlog.Info("Hodor service closed")
	return nil
}