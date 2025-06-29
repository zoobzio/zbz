package core

import (
	"reflect"
	"sync"
	
	"zbz/zlog"
)

// TypeMetadata caches reflection-based metadata for types
type TypeMetadata struct {
	Type         reflect.Type
	IDField      *FieldMetadata
	Fields       map[string]*FieldMetadata
	HasIDField   bool
}

// FieldMetadata contains metadata about a struct field
type FieldMetadata struct {
	Name       string
	FieldIndex int
	Type       reflect.Type
	IsID       bool
	JSONName   string
	Tags       map[string]string
}

// Global metadata cache to avoid duplicate reflection
var (
	metadataCache = make(map[reflect.Type]*TypeMetadata)
	cacheMu       sync.RWMutex
)

// getTypeMetadata returns cached type metadata or computes it
func getTypeMetadata(t reflect.Type) *TypeMetadata {
	// First check cache with read lock
	cacheMu.RLock()
	if cached, exists := metadataCache[t]; exists {
		cacheMu.RUnlock()
		return cached
	}
	cacheMu.RUnlock()

	// Not cached, acquire write lock and compute
	cacheMu.Lock()
	defer cacheMu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := metadataCache[t]; exists {
		return cached
	}

	zlog.Debug("Computing type metadata", 
		zlog.String("type", t.String()),
	)

	// Compute metadata
	metadata := &TypeMetadata{
		Type:   t,
		Fields: make(map[string]*FieldMetadata),
	}

	// Only process struct types
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			
			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			fieldMeta := &FieldMetadata{
				Name:       field.Name,
				FieldIndex: i,
				Type:       field.Type,
				Tags:       make(map[string]string),
			}

			// Parse common tags
			if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
				fieldMeta.JSONName = jsonTag
			} else {
				fieldMeta.JSONName = field.Name
			}

			if zbzTag := field.Tag.Get("zbz"); zbzTag != "" {
				fieldMeta.Tags["zbz"] = zbzTag
				if zbzTag == "id" {
					fieldMeta.IsID = true
					metadata.IDField = fieldMeta
					metadata.HasIDField = true
				}
			}

			if cerealTag := field.Tag.Get("cereal"); cerealTag != "" {
				fieldMeta.Tags["cereal"] = cerealTag
			}

			// Check for common ID field names if no explicit zbz:"id" tag
			if !metadata.HasIDField && isCommonIDFieldName(field.Name) {
				fieldMeta.IsID = true
				metadata.IDField = fieldMeta
				metadata.HasIDField = true
			}

			metadata.Fields[fieldMeta.JSONName] = fieldMeta
		}
	}

	// Cache the computed metadata
	metadataCache[t] = metadata

	zlog.Debug("Type metadata computed and cached",
		zlog.String("type", t.String()),
		zlog.Bool("has_id_field", metadata.HasIDField),
		zlog.Int("field_count", len(metadata.Fields)),
	)

	return metadata
}

// isCommonIDFieldName checks if a field name is commonly used for IDs
func isCommonIDFieldName(name string) bool {
	commonNames := []string{"ID", "Id", "id", "Key", "key", "SKU", "UUID", "Uuid"}
	for _, commonName := range commonNames {
		if name == commonName {
			return true
		}
	}
	return false
}

// GetValue extracts the value of this field from a struct instance
func (fm *FieldMetadata) GetValue(data any) any {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct || fm.FieldIndex >= v.NumField() {
		return nil
	}
	
	field := v.Field(fm.FieldIndex)
	if !field.IsValid() || !field.CanInterface() {
		return nil
	}
	
	return field.Interface()
}

// leverageCerealCache attempts to use cereal's existing field analysis
func leverageCerealCache(t reflect.Type) map[string][]string {
	// Try to access cereal's internal scoping cache
	// This is a simplified implementation - in reality we'd want to
	// extend cereal to expose its field analysis capabilities
	
	// For now, we'll use our own cache but this demonstrates where
	// we would integrate with cereal's existing reflection work
	zlog.Debug("Attempting to leverage cereal cache", 
		zlog.String("type", t.String()),
	)
	
	// Placeholder - would call into cereal's cache system
	return nil
}

// integrateWithCerealSecurity checks if the type has cereal security tags
func integrateWithCerealSecurity(metadata *TypeMetadata) {
	hasSecurityTags := false
	
	for _, field := range metadata.Fields {
		if cerealTag, exists := field.Tags["cereal"]; exists {
			if cerealTag == "encrypt" || cerealTag == "pii" || cerealTag == "secret" {
				hasSecurityTags = true
				break
			}
		}
	}
	
	if hasSecurityTags {
		zlog.Debug("Type has cereal security tags",
			zlog.String("type", metadata.Type.String()),
		)
	}
}

// ClearCache clears the metadata cache (useful for testing)
func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	
	metadataCache = make(map[reflect.Type]*TypeMetadata)
	
	zlog.Debug("Type metadata cache cleared")
}