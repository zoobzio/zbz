package catalog

import (
	"reflect"
)

// PUBLIC API - Only two functions exposed

// Select returns comprehensive metadata for a type
// Handles everything internally: cache check, reflection, storage
// This is the ONLY way to get metadata - always works
func Select[T any]() ModelMetadata {
	var zero T
	t := reflect.TypeOf(zero)
	typeName := getTypeName(t)
	
	// Check cache first
	cacheMutex.RLock()
	if cached, exists := metadataCache[typeName]; exists {
		cacheMutex.RUnlock()
		return cached
	}
	cacheMutex.RUnlock()
	
	// Extract and cache metadata
	metadata := extractMetadata(t, zero)
	
	cacheMutex.Lock()
	metadataCache[typeName] = metadata
	cacheMutex.Unlock()
	
	return metadata
}

// Browse returns all type names that have been registered in the catalog
// Useful for type discovery and debugging
func Browse() []string {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	
	var types []string
	for typeName := range metadataCache {
		types = append(types, typeName)
	}
	return types
}

// GetTypeName returns the string type name for a generic type
// This is the only type name extraction function
func GetTypeName[T any]() string {
	var zero T
	return getTypeName(reflect.TypeOf(zero))
}

// Internal helper functions (not exported)

// ensureMetadata is now incorporated directly into Select[T]()
// getByTypeName is removed - only Select[T]() should be used
// All other convenience functions removed - users extract from Select[T]()