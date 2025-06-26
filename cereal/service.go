package cereal

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// CerealProvider defines the unified interface that all serialization providers implement
type CerealProvider interface {
	// Core serialization operations
	Marshal(data any) ([]byte, error)
	Unmarshal(data []byte, target any) error
	
	// Scoped serialization (no-op if no scope tags exist)
	MarshalScoped(data any, userPermissions []string) ([]byte, error)
	UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error
	
	// Format metadata
	ContentType() string
	Format() string
	
	// Performance and capabilities
	SupportsBinaryData() bool
	SupportsStreaming() bool
	
	// Provider lifecycle
	Close() error
}

// CerealConfig defines provider-agnostic cereal configuration
type CerealConfig struct {
	// Service configuration
	Name           string `yaml:"name" json:"name"`
	DefaultFormat  string `yaml:"default_format" json:"default_format"`     // "json", "raw", "string", "gob"
	
	// Performance settings
	EnableCaching  bool   `yaml:"enable_caching" json:"enable_caching"`     // Cache serialization results
	CacheSize      int    `yaml:"cache_size" json:"cache_size"`             // Max cache entries
	CompressAbove  int    `yaml:"compress_above" json:"compress_above"`     // Compress data above N bytes
	
	// Type-based format selection
	TypeFormats    map[string]string `yaml:"type_formats,omitempty" json:"type_formats,omitempty"`
	
	// Scoped serialization settings
	EnableScoping  bool   `yaml:"enable_scoping" json:"enable_scoping"`     // Enable field-level permissions
	DefaultScope   string `yaml:"default_scope" json:"default_scope"`       // Default permission scope
	StrictMode     bool   `yaml:"strict_mode" json:"strict_mode"`           // Fail on permission violations
	
	// Provider-specific extensions
	Extensions     map[string]interface{} `yaml:"extensions,omitempty" json:"extensions,omitempty"`
}

// DefaultConfig returns sensible defaults for cereal configuration
func DefaultConfig() CerealConfig {
	return CerealConfig{
		Name:          "cereal",
		DefaultFormat: "json",
		EnableCaching: false,
		CacheSize:     1000,
		CompressAbove: 1024 * 1024, // 1MB
		EnableScoping: true,
		StrictMode:    false,
		TypeFormats:   make(map[string]string),
		Extensions:    make(map[string]interface{}),
	}
}

// Operation types for field-level permissions
const (
	OperationRead   = "read"
	OperationWrite  = "write"
	OperationCreate = "create"
	OperationUpdate = "update"
)

// FieldScope represents the scope configuration for a field
type FieldScope struct {
	Read  []string // Required permissions to read this field
	Write []string // Required permissions to write this field
}

// zCereal is the singleton service that orchestrates serialization operations
type zCereal struct {
	provider       CerealProvider // Backend provider wrapper
	config         CerealConfig   // Service configuration
	contractName   string         // Name of the contract that created this singleton
	scopeCache     *scopeCache    // Cached scope metadata for performance
	mu             sync.RWMutex   // Protects provider and config
}

// scopeCache caches reflection-based scope metadata for performance
type scopeCache struct {
	cache map[reflect.Type]map[string]FieldScope
	mu    sync.RWMutex
}

// newScopeCache creates a new scope cache
func newScopeCache() *scopeCache {
	return &scopeCache{
		cache: make(map[reflect.Type]map[string]FieldScope),
	}
}

// getFieldScopes returns cached scope metadata for a type
func (sc *scopeCache) getFieldScopes(t reflect.Type) (map[string]FieldScope, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	scopes, exists := sc.cache[t]
	return scopes, exists
}

// setFieldScopes caches scope metadata for a type
func (sc *scopeCache) setFieldScopes(t reflect.Type, scopes map[string]FieldScope) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cache[t] = scopes
}

// singleton instance
var cereal *zCereal
var cerealMu sync.RWMutex

// configureFromContract initializes the singleton from a contract's registration
func configureFromContract(contractName string, provider CerealProvider, config CerealConfig) error {
	cerealMu.Lock()
	defer cerealMu.Unlock()
	
	// Check if we need to replace existing singleton
	if cereal != nil && cereal.contractName != contractName {
		// Close old provider
		if err := cereal.provider.Close(); err != nil {
			// Log warning but continue
		}
	} else if cereal != nil && cereal.contractName == contractName {
		// Same contract, no need to replace
		return nil
	}

	// Create service singleton
	cereal = &zCereal{
		provider:     provider,
		config:       config,
		contractName: contractName,
		scopeCache:   newScopeCache(),
	}

	return nil
}

// parseFieldScopes extracts scope information from struct tags with caching
func (z *zCereal) parseFieldScopes(t reflect.Type) map[string]FieldScope {
	// Check cache first
	if scopes, exists := z.scopeCache.getFieldScopes(t); exists {
		return scopes
	}
	
	result := make(map[string]FieldScope)
	
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	if t.Kind() != reflect.Struct {
		z.scopeCache.setFieldScopes(t, result)
		return result
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" && jsonTag != "-" {
			fieldName = strings.Split(jsonTag, ",")[0]
		}

		// Parse scope tag
		scopeTag := field.Tag.Get("scope")
		if scopeTag == "" {
			continue // No scope restrictions
		}

		fieldScope := parseScope(scopeTag)
		result[fieldName] = fieldScope
	}

	// Cache the result
	z.scopeCache.setFieldScopes(t, result)
	return result
}

// parseScope parses a scope tag string into FieldScope
func parseScope(scopeTag string) FieldScope {
	scope := FieldScope{}
	
	// Parse scope tag format: "read:users:admin,write:admin"
	parts := strings.Split(scopeTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Determine if this is read or write permission
		if strings.HasPrefix(part, "read:") {
			perm := strings.TrimPrefix(part, "read:")
			scope.Read = append(scope.Read, perm)
		} else if strings.HasPrefix(part, "write:") {
			perm := strings.TrimPrefix(part, "write:")
			scope.Write = append(scope.Write, perm)
		} else {
			// Default to read permission if no prefix
			scope.Read = append(scope.Read, part)
		}
	}

	return scope
}

// hasPermission checks if user has any of the required permissions
func hasPermission(userPermissions []string, requiredPermissions []string) bool {
	if len(requiredPermissions) == 0 {
		return true // No restrictions
	}

	for _, userPerm := range userPermissions {
		for _, reqPerm := range requiredPermissions {
			if userPerm == reqPerm {
				return true
			}
		}
	}
	return false
}

// filterFieldsForRead removes fields that the user doesn't have read access to
func (z *zCereal) filterFieldsForRead(data any, userPermissions []string) any {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return data // Return as-is for non-struct types
	}

	t := val.Type()
	fieldScopes := z.parseFieldScopes(t)
	
	// If no scope restrictions exist, return original data
	if len(fieldScopes) == 0 {
		return data
	}

	// Create a new struct with the same type
	newVal := reflect.New(t).Elem()

	for i := 0; i < val.NumField(); i++ {
		field := t.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" && jsonTag != "-" {
			fieldName = strings.Split(jsonTag, ",")[0]
		}

		// Check if field has scope restrictions
		if fieldScope, hasScope := fieldScopes[fieldName]; hasScope {
			// Check read permission
			if hasPermission(userPermissions, fieldScope.Read) {
				// User has read access, copy the field
				if newVal.Field(i).CanSet() {
					newVal.Field(i).Set(fieldValue)
				}
			}
			// If no read permission, field remains zero value (effectively filtered out)
		} else {
			// No scope restrictions, copy the field
			if newVal.Field(i).CanSet() {
				newVal.Field(i).Set(fieldValue)
			}
		}
	}

	return newVal.Interface()
}

// validateWritePermissions validates write permissions for incoming data
func (z *zCereal) validateWritePermissions(inputFields map[string]any, target any, userPermissions []string, operation string) error {
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct")
	}

	t := val.Type()
	fieldScopes := z.parseFieldScopes(t)
	
	// If no scope restrictions exist, allow all writes
	if len(fieldScopes) == 0 {
		return nil
	}

	// Build field name mapping
	fieldMap := make(map[string]reflect.StructField)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" && jsonTag != "-" {
			fieldName = strings.Split(jsonTag, ",")[0]
		}
		fieldMap[fieldName] = field
	}

	// Check write permissions for each field in the input
	var unauthorized []string
	for fieldName := range inputFields {
		if structField, exists := fieldMap[fieldName]; exists {
			if fieldScope, hasScope := fieldScopes[fieldName]; hasScope {
				// Field has scope restrictions, check write permissions
				if !hasPermission(userPermissions, fieldScope.Write) {
					unauthorized = append(unauthorized, fieldName)
				}
			}
			// If no scope restrictions, allow write
		}
	}

	// If any fields are unauthorized, return error
	if len(unauthorized) > 0 {
		return fmt.Errorf("insufficient permissions to modify fields: %s", strings.Join(unauthorized, ", "))
	}

	return nil
}

// Package-level functions (singleton delegation)

// Marshal serializes data using the configured provider
func Marshal(data any) ([]byte, error) {
	cerealMu.RLock()
	provider := cereal.provider
	cerealMu.RUnlock()
	
	if provider == nil {
		return nil, fmt.Errorf("no cereal provider configured")
	}
	
	return provider.Marshal(data)
}

// Unmarshal deserializes data using the configured provider
func Unmarshal(data []byte, target any) error {
	cerealMu.RLock()
	provider := cereal.provider
	cerealMu.RUnlock()
	
	if provider == nil {
		return fmt.Errorf("no cereal provider configured")
	}
	
	return provider.Unmarshal(data, target)
}

// MarshalScoped serializes data with field-level scope filtering
func MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	cerealMu.RLock()
	provider := cereal.provider
	cerealMu.RUnlock()
	
	if provider == nil {
		return nil, fmt.Errorf("no cereal provider configured")
	}
	
	return provider.MarshalScoped(data, userPermissions)
}

// UnmarshalScoped deserializes data with field-level scope validation
func UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	cerealMu.RLock()
	provider := cereal.provider
	cerealMu.RUnlock()
	
	if provider == nil {
		return fmt.Errorf("no cereal provider configured")
	}
	
	return provider.UnmarshalScoped(data, target, userPermissions, operation)
}

// GetProvider returns the current provider (for advanced usage)
func GetProvider() CerealProvider {
	cerealMu.RLock()
	defer cerealMu.RUnlock()
	
	if cereal == nil {
		return nil
	}
	
	return cereal.provider
}

// GetConfig returns the current configuration
func GetConfig() CerealConfig {
	cerealMu.RLock()
	defer cerealMu.RUnlock()
	
	if cereal == nil {
		return DefaultConfig()
	}
	
	return cereal.config
}