package hodor

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// Public API functions following the zlog/flux pattern
// These work with the global hodor singleton for contract registry

// RegisterContract registers a contract created by providers
func RegisterContract(alias string, provider HodorProvider) error {
	return hodor.RegisterContract(alias, provider)
}

// Unregister removes a contract from the registry
func Unregister(alias string) error {
	return hodor.Unregister(alias)
}

// List returns information about all registered contracts
func List() []ContractInfo {
	return hodor.List()
}

// Status returns the status of a specific contract
func Status(alias string) (ContractStatus, error) {
	return hodor.Status(alias)
}

// Close shuts down all contracts and cleans up
func Close() error {
	return hodor.Close()
}

// Helper functions for working with specific mounts

// GetContract returns contract information
func GetContract(alias string) (ContractInfo, error) {
	// Check if contract exists
	status, err := hodor.Status(alias)
	if err != nil {
		return ContractInfo{}, err
	}
	
	if status.State != ContractStateActive {
		return ContractInfo{}, fmt.Errorf("contract '%s' is not active (state: %s)", alias, status.State)
	}
	
	// Find the contract in hodor's registry
	hodor.mu.RLock()
	contract, exists := hodor.contracts[alias]
	hodor.mu.RUnlock()
	
	if !exists {
		return ContractInfo{}, fmt.Errorf("contract '%s' not found in registry", alias)
	}
	
	return ContractInfo{
		Alias:     contract.alias,
		Provider:  contract.provider.GetProvider(),
		Status:    string(contract.status.State),
		CreatedAt: contract.createdAt,
	}, nil
}

// QuickMemory creates and registers a memory storage contract
func QuickMemory(alias string) (*HodorContract, error) {
	contract := NewMemory(map[string]interface{}{})
	err := contract.Register(alias)
	return contract, err
}

// HodorProviderConfig provides common configuration for all providers
type HodorProviderConfig struct {
	Provider string                 `yaml:"provider"` // s3, gcs, azure, minio, etc.
	Config   map[string]interface{} `yaml:"config"`   // Provider-specific config
}

// RegisterWithYAML registers storage using YAML configuration (legacy)
func RegisterWithYAML(alias string, yamlConfig []byte) (*HodorContract, error) {
	// Parse YAML config
	var config HodorProviderConfig
	err := yaml.Unmarshal(yamlConfig, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}
	
	// Only memory provider supported in legacy mode
	if config.Provider == "memory" {
		contract := NewMemory(config.Config)
		err = contract.Register(alias)
		return contract, err
	}
	
	return nil, fmt.Errorf("provider '%s' not supported in legacy YAML mode", config.Provider)
}

// IsAvailable checks if a provider is available
func IsAvailable(provider string) bool {
	providers := ListStorageProviders()
	for _, p := range providers {
		if p == provider {
			return true
		}
	}
	return false
}

// Providers returns a list of available storage providers
func Providers() []string {
	return ListStorageProviders()
}