package catalog

import (
	"reflect"
)

// User-facing API - always uses generics, never strings
// Lazy extraction happens automatically on first access

// Select returns comprehensive metadata for a type
func Select[T any]() ModelMetadata {
	var zero T
	return ensureMetadata(zero)
}

// GetFields returns just the field metadata for a type
func GetFields[T any]() []FieldMetadata {
	metadata := Select[T]()
	return metadata.Fields
}

// GetScopes returns all scope requirements for a type
func GetScopes[T any]() []string {
	metadata := Select[T]()
	var scopes []string
	for _, field := range metadata.Fields {
		scopes = append(scopes, field.Scopes...)
	}
	
	// Add model-level scopes if type implements ScopeProvider
	for _, function := range metadata.Functions {
		if function.Convention == "ScopeProvider" {
			// Type implements ScopeProvider - could extract model-level scopes here
			break
		}
	}
	
	return uniqueStrings(scopes)
}

// GetEncryptionFields returns fields that require encryption
func GetEncryptionFields[T any]() []FieldMetadata {
	var encryptedFields []FieldMetadata
	for _, field := range GetFields[T]() {
		if field.Encryption.Type != "" {
			encryptedFields = append(encryptedFields, field)
		}
	}
	return encryptedFields
}

// GetValidationFields returns fields with validation rules
func GetValidationFields[T any]() []FieldMetadata {
	var validatedFields []FieldMetadata
	for _, field := range GetFields[T]() {
		if field.Validation.Required || len(field.Validation.CustomRules) > 0 || len(field.Validation.Constraints) > 0 {
			validatedFields = append(validatedFields, field)
		}
	}
	return validatedFields
}

// GetRedactionRules returns field redaction mappings
func GetRedactionRules[T any]() map[string]string {
	rules := make(map[string]string)
	for _, field := range GetFields[T]() {
		if field.Redaction.Value != "" {
			rules[field.Name] = field.Redaction.Value
		}
	}
	return rules
}

// Wrap creates a container around user data
func Wrap[T any](data T) *Container[T] {
	// Ensure metadata exists (triggers extraction)
	Select[T]()
	return NewContainer(data)
}

// HasConvention checks if a type implements a specific convention
func HasConvention[T any](conventionName string) bool {
	metadata := Select[T]()
	for _, function := range metadata.Functions {
		if function.Convention == conventionName {
			return true
		}
	}
	return false
}

// GetTypeName returns the string type name for a generic type
func GetTypeName[T any]() string {
	var zero T
	return getTypeName(reflect.TypeOf(zero))
}

// Internal API for cross-service communication (lowercase = internal)

// getByTypeName allows services to look up metadata by string name
func getByTypeName(typeName string) (ModelMetadata, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	metadata, exists := metadataCache[typeName]
	return metadata, exists
}

// listRegisteredTypes returns all cached type names
func listRegisteredTypes() []string {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	
	var types []string
	for typeName := range metadataCache {
		types = append(types, typeName)
	}
	return types
}

// ensureMetadata performs lazy extraction and caching
func ensureMetadata[T any](example T) ModelMetadata {
	t := reflect.TypeOf(example)
	typeName := getTypeName(t)
	
	// Check cache first
	cacheMutex.RLock()
	if cached, exists := metadataCache[typeName]; exists {
		cacheMutex.RUnlock()
		return cached
	}
	cacheMutex.RUnlock()
	
	// Extract and cache
	metadata := extractMetadata(t, example)
	
	cacheMutex.Lock()
	metadataCache[typeName] = metadata
	cacheMutex.Unlock()
	
	return metadata
}

// Helper functions

func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, str := range slice {
		if !seen[str] && str != "" {
			seen[str] = true
			result = append(result, str)
		}
	}
	
	return result
}

// clearCache clears the metadata cache (useful for testing)
func clearCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	metadataCache = make(map[string]ModelMetadata)
}