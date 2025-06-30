package cereal

import (
	"slices"
	"strings"
)

// DefaultScopingProcessor applies permission-based scoping to fields
func DefaultScopingProcessor(field Field) []Field {
	// Check if user has permission for this field based on catalog scopes
	if hasPermissionForField(field.Metadata.Scopes, field.Permissions) {
		// User has permission - return original field
		return []Field{field}
	}
	
	// User lacks permission - return redacted field
	redactedField := field
	redactedField.Value = getRedactedValueForField(field)
	return []Field{redactedField}
}

// SecurityFieldProcessor handles security-sensitive field types
func SecurityFieldProcessor(field Field) []Field {
	// Always redact security fields unless explicitly permitted
	switch field.Type {
	case SecretType:
		// Secrets are always redacted unless user has 'secret' permission
		if !slices.Contains(field.Permissions, "secret") {
			redactedField := field
			redactedField.Value = "[SECRET_REDACTED]"
			return []Field{redactedField}
		}
		
	case PIIType:
		// PII requires 'pii' permission
		if !slices.Contains(field.Permissions, "pii") {
			redactedField := field
			redactedField.Value = getRedactedPII(field)
			return []Field{redactedField}
		}
		
	case FinancialType:
		// Financial data requires 'financial' permission
		if !slices.Contains(field.Permissions, "financial") {
			redactedField := field
			redactedField.Value = getRedactedFinancial(field)
			return []Field{redactedField}
		}
	}
	
	// Field is permitted or not a security field - return as-is
	return []Field{field}
}

// hasPermissionForField checks if user has permission using scope logic
func hasPermissionForField(fieldScopes []string, userPermissions []string) bool {
	if len(fieldScopes) == 0 {
		return true // No restrictions
	}
	
	// Check each scope group (comma-separated = OR logic)
	for _, scopeGroup := range fieldScopes {
		if satisfiesScopeGroup(scopeGroup, userPermissions) {
			return true // Found one satisfying group
		}
	}
	
	return false // No group satisfied
}

// satisfiesScopeGroup checks if user satisfies a single scope group
func satisfiesScopeGroup(scopeGroup string, userPermissions []string) bool {
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

// getRedactedValueForField creates appropriate redacted value based on field metadata
func getRedactedValueForField(field Field) any {
	// Use catalog redaction info if available
	if field.Metadata.Redaction.Value != "" {
		return field.Metadata.Redaction.Value
	}
	
	// Check validation constraints for appropriate redaction
	if field.Metadata.Validation.Required {
		// Need validation-satisfying redaction
		for _, rule := range field.Metadata.Validation.CustomRules {
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
	
	// Default redaction based on field type
	switch field.Type {
	case StringType:
		return "[REDACTED]"
	case IntType:
		return 0
	case FloatType:
		return 0.0
	case BoolType:
		return false
	case SliceType:
		return []any{}
	case MapType:
		return map[string]any{}
	default:
		return nil
	}
}

// getRedactedPII provides PII-specific redaction patterns
func getRedactedPII(field Field) any {
	for _, rule := range field.Metadata.Validation.CustomRules {
		switch rule {
		case "ssn":
			return "XXX-XX-XXXX"
		case "phone":
			return "XXX-XXX-XXXX"
		case "address":
			return "[ADDRESS_REDACTED]"
		case "email":
			return "redacted@example.com"
		}
	}
	return "[PII_REDACTED]"
}

// getRedactedFinancial provides financial-specific redaction patterns
func getRedactedFinancial(field Field) any {
	for _, rule := range field.Metadata.Validation.CustomRules {
		switch rule {
		case "creditcard":
			return "0000-0000-0000-0000"
		case "bank_account":
			return "XXXXXXXXXXXX"
		case "routing":
			return "XXXXXXXXX"
		}
	}
	return "[FINANCIAL_REDACTED]"
}

// init registers default processors
func init() {
	// Register default scoping processor for all basic types
	RegisterFieldProcessor(StringType, DefaultScopingProcessor)
	RegisterFieldProcessor(IntType, DefaultScopingProcessor)
	RegisterFieldProcessor(FloatType, DefaultScopingProcessor)
	RegisterFieldProcessor(BoolType, DefaultScopingProcessor)
	RegisterFieldProcessor(SliceType, DefaultScopingProcessor)
	RegisterFieldProcessor(MapType, DefaultScopingProcessor)
	RegisterFieldProcessor(StructType, DefaultScopingProcessor)
	
	// Register security processors for sensitive types
	RegisterFieldProcessor(SecretType, SecurityFieldProcessor)
	RegisterFieldProcessor(PIIType, SecurityFieldProcessor)
	RegisterFieldProcessor(FinancialType, SecurityFieldProcessor)
	RegisterFieldProcessor(MedicalType, SecurityFieldProcessor)
}