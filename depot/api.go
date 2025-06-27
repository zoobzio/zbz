package depot

import (
	"time"
)

// Public API functions that delegate to the singleton service (like cache pattern)

// Storage operations that use the global singleton

// Get retrieves data from the singleton storage
func Get(key string) ([]byte, error) {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.provider.Get(key)
}

// Set stores data in the singleton storage
func Set(key string, data []byte, ttl time.Duration) error {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.provider.Set(key, data, ttl)
}

// Delete removes data from the singleton storage
func Delete(key string) error {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.provider.Delete(key)
}

// Exists checks if a key exists in the singleton storage
func Exists(key string) (bool, error) {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.provider.Exists(key)
}

// List returns keys with the given prefix from singleton storage
func List(prefix string) ([]string, error) {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.provider.List(prefix)
}

// Stat returns file information from singleton storage
func Stat(key string) (FileInfo, error) {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.provider.Stat(key)
}

// Subscribe to changes for a specific key in singleton storage
func Subscribe(key string, callback ChangeCallback) (SubscriptionID, error) {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.provider.Subscribe(key, callback)
}

// Unsubscribe from changes in singleton storage
func Unsubscribe(id SubscriptionID) error {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.provider.Unsubscribe(id)
}

// Metadata operations using cereal serialization

// SetWithMetadata stores data with structured metadata using cereal
func SetWithMetadata(key string, data []byte, metadata interface{}) error {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.setWithMetadata(key, data, metadata)
}

// GetWithMetadata retrieves data and deserializes metadata using cereal
func GetWithMetadata(key string, metadata interface{}) ([]byte, error) {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.getWithMetadata(key, metadata)
}

// SetJSON stores a JSON object using cereal serialization
func SetJSON(key string, object interface{}) error {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.setJSON(key, object)
}

// GetJSON retrieves and deserializes a JSON object using cereal
func GetJSON(key string, target interface{}) error {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.getJSON(key, target)
}

// SetJSONScoped stores a JSON object with field-level scoping
func SetJSONScoped(key string, object interface{}, userPermissions []string) error {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.setJSONScoped(key, object, userPermissions)
}

// GetJSONScoped retrieves and deserializes a JSON object with scoping
func GetJSONScoped(key string, target interface{}, userPermissions []string) error {
	if depot == nil {
		panic("depot not configured - create and register a contract first")
	}
	return depot.getJSONScoped(key, target, userPermissions)
}

// Service management functions

// Provider returns the current depot provider
func Provider() DepotProvider {
	if depot == nil {
		return nil
	}
	return depot.provider
}

// Config returns the current depot configuration
func Config() DepotConfig {
	if depot == nil {
		return DepotConfig{}
	}
	return depot.config
}

// IsConfigured returns true if the depot service has been configured
func IsConfigured() bool {
	return depot != nil
}

// Close shuts down the depot service
func Close() error {
	if depot == nil {
		return nil
	}
	return depot.Close()
}