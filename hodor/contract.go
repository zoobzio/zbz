package hodor

import (
	"fmt"
	"time"
)

// HodorContract represents a storage bucket with reactive capabilities
// Unlike zlog contracts that compete for singleton, hodor contracts are independent instances
type HodorContract struct {
	name     string        // Contract name
	provider HodorProvider // Standardized interface for service interaction
	
	// Registration state (if registered with service)
	alias      string
	registered bool
}

// NewContract creates a new hodor contract with provider
func NewContract(name string, provider HodorProvider) *HodorContract {
	return &HodorContract{
		name:       name,
		provider:   provider,
		registered: false,
	}
}

// Register registers this storage contract with the global hodor service for discovery
func (c *HodorContract) Register(alias string) error {
	if c.registered {
		return fmt.Errorf("contract already registered as '%s'", c.alias)
	}
	
	// Register with global hodor service
	err := hodor.RegisterContract(alias, c.provider)
	if err != nil {
		return fmt.Errorf("failed to register contract: %w", err)
	}
	
	c.alias = alias
	c.registered = true
	
	return nil
}

// Provider returns the provider interface for direct access to the storage backend
func (c *HodorContract) Provider() HodorProvider {
	return c.provider
}

// Name returns the contract name
func (c *HodorContract) Name() string {
	return c.name
}

// Alias returns the mount alias (if mounted)
func (c *HodorContract) Alias() string {
	return c.alias
}


// GetProvider returns the provider name
func (c *HodorContract) GetProvider() string {
	return c.provider.GetProvider()
}

// Storage operations - proxy through provider interface
func (c *HodorContract) Get(key string) ([]byte, error) {
	return c.provider.Get(key)
}

func (c *HodorContract) Set(key string, data []byte, ttl time.Duration) error {
	return c.provider.Set(key, data, ttl)
}

func (c *HodorContract) Delete(key string) error {
	return c.provider.Delete(key)
}

func (c *HodorContract) Exists(key string) (bool, error) {
	return c.provider.Exists(key)
}

func (c *HodorContract) List(prefix string) ([]string, error) {
	return c.provider.List(prefix)
}

func (c *HodorContract) Stat(key string) (FileInfo, error) {
	return c.provider.Stat(key)
}

// Subscribe to changes for a specific key
func (c *HodorContract) Subscribe(key string, callback ChangeCallback) (SubscriptionID, error) {
	return c.provider.Subscribe(key, callback)
}

// Unsubscribe from changes
func (c *HodorContract) Unsubscribe(id SubscriptionID) error {
	return c.provider.Unsubscribe(id)
}

// Registration management operations
func (c *HodorContract) Status() (ContractStatus, error) {
	if !c.registered {
		return ContractStatus{}, fmt.Errorf("contract not registered")
	}
	return hodor.Status(c.alias)
}

func (c *HodorContract) Unregister() error {
	if !c.registered {
		return fmt.Errorf("contract not registered")
	}
	
	err := hodor.Unregister(c.alias)
	if err != nil {
		return err
	}
	
	c.registered = false
	c.alias = ""
	
	return nil
}