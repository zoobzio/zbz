package catalog

import (
	"fmt"
	"strings"
)

// Example showing how other zbz framework packages would integrate with catalog

// CerealIntegration demonstrates how the cereal package would use catalog

// GetScopedFields simulates how cereal would get fields requiring specific scopes
func GetScopedFields[T any](userPermissions []string) []FieldMetadata {
	var scopedFields []FieldMetadata
	
	// Get all fields with scopes using catalog
	fields := GetFields[T]()
	
	for _, field := range fields {
		if len(field.Scopes) > 0 {
			// Check if user has required permissions
			hasAccess := false
			for _, scope := range field.Scopes {
				for _, userPerm := range userPermissions {
					if userPerm == scope {
						hasAccess = true
						break
					}
				}
			}
			
			if !hasAccess {
				// Field needs redaction - catalog provides the redaction value
				scopedFields = append(scopedFields, field)
			}
		}
	}
	
	return scopedFields
}

// GetRedactionPlan simulates how cereal would build a redaction plan
func GetRedactionPlan[T any](userPermissions []string) map[string]string {
	plan := make(map[string]string)
	
	// Get redaction rules from catalog
	redactionRules := GetRedactionRules[T]()
	scopedFields := GetScopedFields[T](userPermissions)
	
	for _, field := range scopedFields {
		if redactionValue, exists := redactionRules[field.Name]; exists {
			plan[field.Name] = redactionValue
		} else {
			// Default redaction if none specified
			plan[field.Name] = "[REDACTED]"
		}
	}
	
	return plan
}

// ValidationIntegration demonstrates how a validation package would use catalog

// GetValidationRules simulates how validation package would extract rules
func GetValidationRules[T any]() map[string][]string {
	rules := make(map[string][]string)
	
	// Get validated fields from catalog
	validatedFields := GetValidationFields[T]()
	
	for _, field := range validatedFields {
		var fieldRules []string
		
		if field.Validation.Required {
			fieldRules = append(fieldRules, "required")
		}
		
		// Add custom rules
		fieldRules = append(fieldRules, field.Validation.CustomRules...)
		
		// Add constraint rules
		for constraint, value := range field.Validation.Constraints {
			fieldRules = append(fieldRules, fmt.Sprintf("%s=%s", constraint, value))
		}
		
		if len(fieldRules) > 0 {
			rules[field.Name] = fieldRules
		}
	}
	
	return rules
}

// HTTPIntegration demonstrates how HTTP package would use catalog for OpenAPI

// GenerateOpenAPISchema simulates OpenAPI schema generation
func GenerateOpenAPISchema[T any]() map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}
	
	properties := schema["properties"].(map[string]interface{})
	required := []string{}
	
	// Get fields from catalog
	fields := GetFields[T]()
	
	for _, field := range fields {
		fieldSchema := map[string]interface{}{
			"type": mapGoTypeToJSONType(field.Type),
		}
		
		// Add description if available
		if field.Description != "" {
			fieldSchema["description"] = field.Description
		}
		
		// Add example if available
		if field.Example != nil {
			fieldSchema["example"] = field.Example
		}
		
		// Add security information for documentation
		if len(field.Scopes) > 0 {
			fieldSchema["x-required-scopes"] = field.Scopes
		}
		
		if field.Encryption.Type != "" {
			fieldSchema["x-encryption"] = field.Encryption.Type
		}
		
		// Add to required list if needed
		if field.Validation.Required {
			required = append(required, field.JSONName)
		}
		
		properties[field.JSONName] = fieldSchema
	}
	
	if len(required) > 0 {
		schema["required"] = required
	}
	
	return schema
}

// DatabaseIntegration demonstrates how database package would use catalog

// GenerateTableSchema simulates database schema generation
func GenerateTableSchema[T any]() string {
	var parts []string
	typeName := GetTypeName[T]()
	
	parts = append(parts, fmt.Sprintf("CREATE TABLE %s (", strings.ToLower(typeName)))
	
	// Standard container fields
	parts = append(parts, "  id VARCHAR(255) PRIMARY KEY,")
	parts = append(parts, "  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,")
	parts = append(parts, "  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,")
	parts = append(parts, "  version INTEGER DEFAULT 1,")
	
	// Get fields from catalog
	fields := GetFields[T]()
	
	for _, field := range fields {
		columnDef := fmt.Sprintf("  %s %s", 
			field.DBColumn, 
			mapGoTypeToSQLType(field.Type))
		
		// Add constraints based on validation
		if field.Validation.Required {
			columnDef += " NOT NULL"
		}
		
		// Add encryption note as comment
		if field.Encryption.Type != "" {
			columnDef += fmt.Sprintf(" -- ENCRYPT:%s", field.Encryption.Type)
		}
		
		parts = append(parts, columnDef+",")
	}
	
	// Remove last comma and close
	if len(parts) > 0 {
		parts[len(parts)-1] = strings.TrimSuffix(parts[len(parts)-1], ",")
	}
	parts = append(parts, ");")
	
	return strings.Join(parts, "\n")
}

// Helper functions for type mapping

func mapGoTypeToJSONType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

func mapGoTypeToSQLType(goType string) string {
	switch goType {
	case "string":
		return "VARCHAR(255)"
	case "int", "int32":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "float32", "float64":
		return "DECIMAL(10,2)"
	case "bool":
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

// ExampleIntegrations shows how to use the integration functions
// Note: This would normally use a concrete type - moved to test file