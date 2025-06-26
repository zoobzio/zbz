package hodor

import (
	"time"
)

// Public API functions that delegate to the singleton service (like cache pattern)

// Storage operations that use the global singleton

// Get retrieves data from the singleton storage
func Get(key string) ([]byte, error) {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.provider.Get(key)
}

// Set stores data in the singleton storage
func Set(key string, data []byte, ttl time.Duration) error {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.provider.Set(key, data, ttl)
}

// Delete removes data from the singleton storage
func Delete(key string) error {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.provider.Delete(key)
}

// Exists checks if a key exists in the singleton storage
func Exists(key string) (bool, error) {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.provider.Exists(key)
}

// List returns keys with the given prefix from singleton storage
func List(prefix string) ([]string, error) {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.provider.List(prefix)
}

// Stat returns file information from singleton storage
func Stat(key string) (FileInfo, error) {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.provider.Stat(key)
}

// Subscribe to changes for a specific key in singleton storage
func Subscribe(key string, callback ChangeCallback) (SubscriptionID, error) {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.provider.Subscribe(key, callback)
}

// Unsubscribe from changes in singleton storage
func Unsubscribe(id SubscriptionID) error {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.provider.Unsubscribe(id)
}

// Metadata operations using cereal serialization

// SetWithMetadata stores data with structured metadata using cereal
func SetWithMetadata(key string, data []byte, metadata interface{}) error {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.setWithMetadata(key, data, metadata)
}

// GetWithMetadata retrieves data and deserializes metadata using cereal
func GetWithMetadata(key string, metadata interface{}) ([]byte, error) {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.getWithMetadata(key, metadata)
}

// SetJSON stores a JSON object using cereal serialization
func SetJSON(key string, object interface{}) error {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.setJSON(key, object)
}

// GetJSON retrieves and deserializes a JSON object using cereal
func GetJSON(key string, target interface{}) error {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.getJSON(key, target)
}

// SetJSONScoped stores a JSON object with field-level scoping
func SetJSONScoped(key string, object interface{}, userPermissions []string) error {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.setJSONScoped(key, object, userPermissions)
}

// GetJSONScoped retrieves and deserializes a JSON object with scoping
func GetJSONScoped(key string, target interface{}, userPermissions []string) error {
	if hodor == nil {
		panic("hodor not configured - create and register a contract first")
	}
	return hodor.getJSONScoped(key, target, userPermissions)
}

// Service management functions

// Provider returns the current hodor provider
func Provider() HodorProvider {
	if hodor == nil {
		return nil
	}
	return hodor.provider
}

// Config returns the current hodor configuration
func Config() HodorConfig {
	if hodor == nil {
		return HodorConfig{}
	}
	return hodor.config
}

// IsConfigured returns true if the hodor service has been configured
func IsConfigured() bool {
	return hodor != nil
}

// Close shuts down the hodor service
func Close() error {
	if hodor == nil {
		return nil
	}
	return hodor.Close()
}