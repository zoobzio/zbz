package cereal

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"zbz/catalog"
)

// CatalogScoper handles scoping logic using catalog metadata instead of reflection
type CatalogScoper struct {
	cache sync.Map // Cache for performance
}

// NewCatalogScoper creates a new catalog-based scoper
func NewCatalogScoper() *CatalogScoper {
	return &CatalogScoper{}
}

// FilterForMarshal applies scoping for marshal operations using catalog metadata
func (cs *CatalogScoper) FilterForMarshal(data any, userPermissions []string) any {
	// Use catalog to get type metadata instead of reflection
	metadata := catalog.ExtractAndCacheMetadata(data)
	
	// Check for convention-based scope requirements first
	if err := cs.checkConventionScopes(data, userPermissions, metadata); err != nil {
		// Return nil or error marker for failed scope check
		return nil
	}
	
	// Create fields for processing based on catalog metadata
	var processedFields []Field
	
	// Process each field using catalog metadata
	for _, fieldMeta := range metadata.Fields {
		// Create field from catalog metadata
		field := Field{
			Key:         fieldMeta.Name,
			Type:        cs.mapCatalogFieldType(fieldMeta),
			Value:       nil, // Will be set by field processor
			Permissions: userPermissions,
			Metadata:    fieldMeta,
		}
		
		// Apply field processor for scoping decision
		processed := ProcessField(field)
		processedFields = append(processedFields, processed...)
	}
	
	// Convert processed fields back to struct format
	return cs.fieldsToStruct(processedFields, metadata)
}

// ValidateUnmarshalPermissions filters unmarshaled data using catalog metadata
func (cs *CatalogScoper) ValidateUnmarshalPermissions(data any, userPermissions []string) error {
	metadata := catalog.ExtractAndCacheMetadata(data)
	
	// Create fields for processing based on catalog metadata
	for _, fieldMeta := range metadata.Fields {
		// Check if user has permission for this field
		if !cs.hasFieldPermission(fieldMeta.Scopes, userPermissions) {
			// User doesn't have permission - emit event for audit trail
			emitFieldScopeEvent(metadata.TypeName, fieldMeta.Name, fieldMeta.Type, userPermissions, false)
		} else {
			// User has permission - emit success event
			emitFieldScopeEvent(metadata.TypeName, fieldMeta.Name, fieldMeta.Type, userPermissions, true)
		}
	}
	
	return nil
}

// hasFieldPermission checks if user has permission for field scopes
// Supports complex scoping like "admin+pii,compliance"
func (cs *CatalogScoper) hasFieldPermission(fieldScopes []string, userPermissions []string) bool {
	if len(fieldScopes) == 0 {
		return true // No restrictions
	}
	
	// Check each scope group (comma-separated = OR logic)
	for _, scopeGroup := range fieldScopes {
		if cs.satisfiesScopeGroup(scopeGroup, userPermissions) {
			return true // Found one satisfying group
		}
	}
	
	return false // No group satisfied
}

// satisfiesScopeGroup checks if user satisfies a single scope group
// scopeGroup: "admin+pii+security" or "compliance"
func (cs *CatalogScoper) satisfiesScopeGroup(scopeGroup string, userPermissions []string) bool {
	// Split on '+' to get individual required permissions (AND logic)
	requiredPerms := strings.Split(scopeGroup, "+")
	
	// User must have ALL permissions in this group
	for _, requiredPerm := range requiredPerms {
		requiredPerm = strings.TrimSpace(requiredPerm)
		if requiredPerm == "" {
			continue // Skip empty parts
		}
		
		if !slices.Contains(userPermissions, requiredPerm) {
			return false // Missing required permission
		}
	}
	
	return true // User has all required permissions
}

// checkConventionScopes checks convention-based scoping (ScopeProvider interface)
func (cs *CatalogScoper) checkConventionScopes(data any, userPermissions []string, metadata catalog.ModelMetadata) error {
	// Check if type implements ScopeProvider convention
	for _, function := range metadata.Functions {
		if function.Convention == "ScopeProvider" {
			// Type implements ScopeProvider - check if it's satisfied
			if scopeProvider, ok := data.(interface{ GetRequiredScopes() []string }); ok {
				requiredScopes := scopeProvider.GetRequiredScopes()
				for _, required := range requiredScopes {
					if !slices.Contains(userPermissions, required) {
						return fmt.Errorf("insufficient scope: requires %s", required)
					}
				}
			}
		}
	}
	return nil
}

// getRedactedFieldValue creates redacted value using field processors
func (cs *CatalogScoper) getRedactedFieldValue(fieldMeta catalog.FieldMetadata, originalValue any) any {
	// Create field for processing
	field := Field{
		Key:      fieldMeta.Name,
		Type:     cs.mapCatalogFieldType(fieldMeta),
		Value:    originalValue,
		Metadata: fieldMeta,
	}
	
	// Apply field processor for redaction
	processedFields := ProcessField(field)
	if len(processedFields) > 0 {
		return processedFields[0].Value
	}
	
	// No processor - return default redaction
	return cs.getDefaultRedaction(fieldMeta)
}

// mapCatalogFieldType maps catalog field metadata to FieldType, considering security annotations
func (cs *CatalogScoper) mapCatalogFieldType(fieldMeta catalog.FieldMetadata) FieldType {
	// Check for security-specific field types first
	if fieldMeta.Encryption.Type != "" {
		switch fieldMeta.Encryption.Type {
		case "pii":
			return PIIType
		case "financial":
			return FinancialType
		case "medical":
			return MedicalType
		}
	}
	
	// Check validation rules for security field types
	for _, rule := range fieldMeta.Validation.CustomRules {
		switch rule {
		case "ssn", "phone", "address":
			return PIIType
		case "creditcard", "bank_account":
			return FinancialType
		case "password", "api_key", "secret":
			return SecretType
		}
	}
	
	// Fall back to basic type mapping
	switch {
	case strings.Contains(fieldMeta.Type, "string"):
		return StringType
	case strings.Contains(fieldMeta.Type, "int"):
		return IntType
	case strings.Contains(fieldMeta.Type, "float"):
		return FloatType
	case strings.Contains(fieldMeta.Type, "bool"):
		return BoolType
	case strings.Contains(fieldMeta.Type, "[]"):
		return SliceType
	case strings.Contains(fieldMeta.Type, "map"):
		return MapType
	default:
		return StructType
	}
}

// fieldsToStruct converts processed fields back to map for serialization
func (cs *CatalogScoper) fieldsToStruct(fields []Field, metadata catalog.ModelMetadata) map[string]any {
	result := make(map[string]any)
	
	for _, field := range fields {
		// Find the corresponding field metadata for JSON name
		jsonName := field.Key
		for _, fieldMeta := range metadata.Fields {
			if fieldMeta.Name == field.Key {
				if fieldMeta.JSONName != "" {
					jsonName = fieldMeta.JSONName
				}
				break
			}
		}
		
		result[jsonName] = field.Value
	}
	
	return result
}

// getDefaultRedaction provides fallback redaction values
func (cs *CatalogScoper) getDefaultRedaction(fieldMeta catalog.FieldMetadata) any {
	// Use catalog redaction info if available
	if fieldMeta.Redaction.Value != "" {
		return fieldMeta.Redaction.Value
	}
	
	// Check validation constraints for appropriate redaction
	if fieldMeta.Validation.Required {
		// Need validation-satisfying redaction
		for _, rule := range fieldMeta.Validation.CustomRules {
			switch rule {
			case "email":
				return "redacted@example.com"
			case "uuid":
				return "00000000-0000-0000-0000-000000000000"
			case "phone":
				return "XXX-XXX-XXXX"
			case "ssn":
				return "XXX-XX-XXXX"
			case "creditcard":
				return "0000-0000-0000-0000"
			}
		}
	}
	
	// Default redaction based on type
	switch cs.mapCatalogFieldType(fieldMeta) {
	case StringType:
		return "[REDACTED]"
	case IntType:
		return 0
	case FloatType:
		return 0.0
	case BoolType:
		return false
	default:
		return nil
	}
}


// Global catalog scoper instance
var catalogScoper = NewCatalogScoper()

// Public API functions that use catalog instead of reflection

// FilterByPermissions applies permission-based filtering using catalog metadata
func FilterByPermissions(data any, permissions []string) (map[string]any, error) {
	result := catalogScoper.FilterForMarshal(data, permissions)
	if result == nil {
		return nil, fmt.Errorf("permission denied or invalid data")
	}
	if resultMap, ok := result.(map[string]any); ok {
		return resultMap, nil
	}
	return nil, fmt.Errorf("unexpected result type from filter")
}

// ValidatePermissions validates that input data doesn't violate permission constraints
func ValidatePermissions(input any, permissions []string) error {
	return catalogScoper.ValidateUnmarshalPermissions(input, permissions)
}