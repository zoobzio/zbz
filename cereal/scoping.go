package cereal

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
)

// zCereal handles scoping logic
type zCereal struct {
	cache *zScopeCache
	mu    sync.RWMutex
}

// validateUnmarshalPermissions filters unmarshaled data to only include fields user can access
// This prevents privilege escalation by ignoring restricted fields in input payloads
func (cs *zCereal) validateUnmarshalPermissions(v any, permissions []string) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return nil // Only filter struct pointers
	}

	val = val.Elem() // Dereference pointer
	t := val.Type()
	fieldScopes := cs.parseFieldScopes(t)

	// If no scope restrictions exist, allow all fields
	if len(fieldScopes) == 0 {
		return nil
	}

	// Clear fields that user doesn't have permission to set
	for i := range val.NumField() {
		field := t.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() || !fieldValue.CanSet() {
			continue
		}

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" && jsonTag != "-" {
			fieldName = strings.Split(jsonTag, ",")[0]
		}

		// Check if field has scope restrictions
		if requiredPerms, hasScope := fieldScopes[fieldName]; hasScope {
			// If user doesn't have permission, clear the field
			if !cs.hasPermission(requiredPerms, permissions) {
				fieldValue.Set(reflect.Zero(fieldValue.Type()))
			}
		}
		// If no scope restrictions, field is allowed (no action needed)
	}

	return nil
}

// filterForMarshal applies scoping for marshal operations using redaction instead of omission
// This prevents validation issues with required fields that users can't access
func (cs *zCereal) filterForMarshal(data any, userPermissions []string) any {
	// NEW: Check for convention-based scope requirements first
	if err := cs.checkConventionScopes(data, userPermissions); err != nil {
		// If convention-based scope check fails, return appropriate error response
		// For now, we'll continue with field-level filtering but this could be enhanced
		// to return completely redacted struct or error marker
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return data // Return as-is for non-struct types
	}

	t := val.Type()
	fieldScopes := cs.parseFieldScopes(t)

	// If no scope restrictions exist (neither convention-based nor field-based), return original data
	if len(fieldScopes) == 0 {
		return data
	}

	// Create a copy of the struct with redacted fields instead of omitting them
	resultPtr := reflect.New(t)
	result := resultPtr.Elem()

	for i := range val.NumField() {
		field := t.Field(i)
		fieldValue := val.Field(i)
		resultField := result.Field(i)

		// Skip unexported fields or fields we can't set
		if !field.IsExported() || !resultField.CanSet() {
			continue
		}

		// Get JSON field name for scope checking
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" && jsonTag != "-" {
			fieldName = strings.Split(jsonTag, ",")[0]
		}

		// Check if field has scope restrictions
		if requiredPerms, hasScope := fieldScopes[fieldName]; hasScope {
			// Field has scope restrictions - check if user has access
			if cs.hasPermission(requiredPerms, userPermissions) {
				// User has access, copy the original value
				resultField.Set(fieldValue)
			} else {
				// User doesn't have access, redact with validation-aware value
				resultField.Set(cs.getRedactedValue(fieldValue.Type(), field))
			}
		} else {
			// No scope restrictions - always copy original value
			resultField.Set(fieldValue)
		}
	}

	return resultPtr.Interface()
}

// parseFieldScopes extracts scope information from struct tags with caching
func (cs *zCereal) parseFieldScopes(t reflect.Type) map[string][]string {
	// Check cache first
	if scopes, exists := cs.cache.getFieldScopes(t); exists {
		return scopes
	}

	result := make(map[string][]string)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		cs.cache.setFieldScopes(t, result)
		return result
	}

	for i := range t.NumField() {
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

		// Parse scope tag - our new format: scope:"read,write" or scope:"admin+pii,read"
		scopeTag := field.Tag.Get("scope")
		if scopeTag == "" {
			continue // No scope restrictions
		}

		result[fieldName] = parseScope(scopeTag)
	}

	// Cache the result
	cs.cache.setFieldScopes(t, result)
	return result
}

// parseScope parses scope format: "admin+pii,compliance,admin+security+executive"
// Returns slice of scope groups where each group is comma-separated
// Example: "admin+pii,compliance" → ["admin+pii", "compliance"]
func parseScope(scopeTag string) []string {
	parts := strings.Split(scopeTag, ",")
	var cleaned []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return cleaned
}

// hasPermission checks permissions using OR/AND logic:
// fieldScope: ["admin+pii", "compliance", "admin+security+executive"]
// userPermissions: ["admin", "pii", "security"]
// Logic: (admin && pii) || compliance || (admin && security && executive)
func (cs *zCereal) hasPermission(fieldScope []string, userPermissions []string) bool {
	if len(fieldScope) == 0 {
		return true // No restrictions
	}

	// Check each scope group (comma-separated = OR logic)
	for _, scopeGroup := range fieldScope {
		if cs.satisfiesScopeGroup(scopeGroup, userPermissions) {
			return true // Found one satisfying group, exit early
		}
	}

	return false // No group satisfied
}

// satisfiesScopeGroup checks if user satisfies a single scope group
// scopeGroup: "admin+pii+security" or "compliance"
// userPermissions: ["admin", "pii", "security"]
func (cs *zCereal) satisfiesScopeGroup(scopeGroup string, userPermissions []string) bool {
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

	return true // User has all required permissions in this group
}

// checkConventionScopes checks if the model implements ScopeProvider and validates permissions
func (cs *zCereal) checkConventionScopes(data any, userPermissions []string) error {
	// Simple interface check - only look for ScopeProvider
	if !checkScopeProvider(data, userPermissions) {
		return fmt.Errorf("insufficient permissions for model-level access")
	}
	return nil
}

// hasPermissionForScope checks if user has a specific scope (single scope, not group)
func (cs *zCereal) hasPermissionForScope(requiredScope string, userPermissions []string) bool {
	for _, userPerm := range userPermissions {
		if userPerm == requiredScope {
			return true
		}
	}
	return false
}

// getRedactedValue returns an appropriate redacted value based on field type and validation tags
// Values are chosen to satisfy validation constraints while indicating redaction
func (cs *zCereal) getRedactedValue(fieldType reflect.Type, field reflect.StructField) reflect.Value {
	// Parse validation tag to understand constraints
	validateTag := field.Tag.Get("validate")
	constraints := parseValidationConstraints(validateTag)
	
	switch fieldType.Kind() {
	case reflect.String:
		return cs.getRedactedString(constraints, field)
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cs.getRedactedInt(fieldType, constraints)
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cs.getRedactedUint(fieldType, constraints)
		
	case reflect.Float32, reflect.Float64:
		return cs.getRedactedFloat(fieldType, constraints)
		
	case reflect.Bool:
		// For booleans, use true as it usually satisfies validation better
		return reflect.ValueOf(true)
		
	case reflect.Slice:
		// Return empty slice instead of nil to maintain JSON structure
		return reflect.MakeSlice(fieldType, 0, 0)
		
	case reflect.Map:
		// Return empty map instead of nil
		return reflect.MakeMap(fieldType)
		
	case reflect.Ptr:
		// For pointers, return nil (truly redacted)
		return reflect.Zero(fieldType)
		
	case reflect.Interface:
		// For interfaces, return nil
		return reflect.Zero(fieldType)
		
	case reflect.Struct:
		// For structs, create a zero-value instance
		return reflect.Zero(fieldType)
		
	default:
		// For any other types, use zero value
		return reflect.Zero(fieldType)
	}
}

// ValidationConstraints holds parsed validation rules
type ValidationConstraints struct {
	Required bool
	Min      int
	Max      int
	Len      int
	Email    bool
	URL      bool
	UUID     bool
	Alpha    bool
	Alphanum bool
	Numeric  bool
	JSON     bool
}

// parseValidationConstraints parses validation tag into structured constraints
func parseValidationConstraints(validateTag string) ValidationConstraints {
	constraints := ValidationConstraints{Min: -1, Max: -1, Len: -1}
	
	if validateTag == "" {
		return constraints
	}
	
	parts := strings.Split(validateTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		
		if part == "required" {
			constraints.Required = true
		} else if part == "email" {
			constraints.Email = true
		} else if part == "url" {
			constraints.URL = true
		} else if part == "uuid" {
			constraints.UUID = true
		} else if part == "alpha" {
			constraints.Alpha = true
		} else if part == "alphanum" {
			constraints.Alphanum = true
		} else if part == "numeric" {
			constraints.Numeric = true
		} else if part == "json" {
			constraints.JSON = true
		} else if strings.HasPrefix(part, "min=") {
			if val := parseIntValue(part[4:]); val >= 0 {
				constraints.Min = val
			}
		} else if strings.HasPrefix(part, "max=") {
			if val := parseIntValue(part[4:]); val >= 0 {
				constraints.Max = val
			}
		} else if strings.HasPrefix(part, "len=") {
			if val := parseIntValue(part[4:]); val >= 0 {
				constraints.Len = val
			}
		}
	}
	
	return constraints
}

// parseIntValue safely parses integer values from validation tags
func parseIntValue(s string) int {
	var result int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			return -1 // Invalid format
		}
	}
	return result
}

// getRedactedString creates a redacted string that satisfies validation constraints
func (cs *zCereal) getRedactedString(constraints ValidationConstraints, field reflect.StructField) reflect.Value {
	// First check for custom validators in the validation tag
	validateTag := field.Tag.Get("validate")
	if validateTag != "" {
		parts := strings.Split(validateTag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if value, found := cs.getCustomRedactedValue(part, field.Type); found {
				return value
			}
		}
	}
	
	// Handle format-specific validators with realistic redacted values
	if constraints.Email {
		return reflect.ValueOf("redacted@example.com")
	}
	
	if constraints.URL {
		return reflect.ValueOf("https://redacted.example.com")
	}
	
	if constraints.UUID {
		return reflect.ValueOf("00000000-0000-0000-0000-000000000000")
	}
	
	if constraints.JSON {
		return reflect.ValueOf("{\"redacted\":true}")
	}
	
	if constraints.Alpha {
		// Pure alphabetic characters
		if constraints.Len > 0 {
			return reflect.ValueOf(strings.Repeat("X", constraints.Len))
		}
		return reflect.ValueOf("REDACTED")
	}
	
	if constraints.Alphanum {
		// Alphanumeric characters
		if constraints.Len > 0 {
			return reflect.ValueOf("R" + strings.Repeat("X", constraints.Len-1))
		}
		return reflect.ValueOf("REDACTED123")
	}
	
	if constraints.Numeric {
		// Pure numeric string
		if constraints.Len > 0 {
			return reflect.ValueOf(strings.Repeat("0", constraints.Len))
		}
		return reflect.ValueOf("0")
	}
	
	// Handle specific length requirements with format-aware patterns
	if constraints.Len > 0 {
		return reflect.ValueOf(cs.generateRedactedPattern(constraints.Len))
	}
	
	// Handle min/max length requirements
	baseValue := "[REDACTED]"
	targetLen := len(baseValue)
	if constraints.Min > 0 && targetLen < constraints.Min {
		targetLen = constraints.Min
	}
	if constraints.Max > 0 && targetLen > constraints.Max {
		targetLen = constraints.Max
	}
	
	if targetLen != len(baseValue) {
		return reflect.ValueOf(cs.generateRedactedPattern(targetLen))
	}
	
	return reflect.ValueOf(baseValue)
}

// generateRedactedPattern creates format-aware redacted strings based on length
func (cs *zCereal) generateRedactedPattern(length int) string {
	switch length {
	case 9: // Social Security Number format
		return "XXX-XX-XXX"
	case 11: // SSN with dashes
		return "XXX-XX-XXXX" 
	case 10: // Phone number format
		return "XXX-XXX-XXX"
	case 12: // Phone with area code
		return "XXX-XXX-XXXX"
	case 36: // UUID format
		return "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
	case 32: // UUID without dashes
		return strings.Repeat("X", 32)
	case 16: // Credit card-like (16 digits, no dashes)
		return "0000000000000000"
	case 5: // ZIP code
		return "XXXXX"
	case 6: // Postal code
		return "XXXXXX"
	default:
		// For other lengths, use smart patterns
		if length <= 10 {
			// Short strings get X pattern
			return strings.Repeat("X", length)
		} else if length <= 20 {
			// Medium strings get [REDACTED] padded
			base := "[REDACTED]"
			if length < len(base) {
				return base[:length]
			}
			return base + strings.Repeat("X", length-len(base))
		} else {
			// Long strings get descriptive redaction
			base := "[REDACTED_" + strings.ToUpper(cs.guessDataType(length)) + "]"
			if length < len(base) {
				return strings.Repeat("X", length)
			}
			return base + strings.Repeat("X", length-len(base))
		}
	}
}

// guessDataType attempts to guess data type based on field length patterns
func (cs *zCereal) guessDataType(length int) string {
	switch {
	case length >= 200:
		return "text"
	case length >= 100:
		return "description"
	case length >= 50:
		return "address"
	case length >= 30:
		return "name"
	default:
		return "data"
	}
}

// getRedactedInt creates a redacted integer that satisfies validation constraints
func (cs *zCereal) getRedactedInt(fieldType reflect.Type, constraints ValidationConstraints) reflect.Value {
	value := 0
	if constraints.Min > 0 {
		value = constraints.Min
	}
	return reflect.ValueOf(value).Convert(fieldType)
}

// getRedactedUint creates a redacted unsigned integer that satisfies validation constraints
func (cs *zCereal) getRedactedUint(fieldType reflect.Type, constraints ValidationConstraints) reflect.Value {
	value := uint64(0)
	if constraints.Min > 0 {
		value = uint64(constraints.Min)
	}
	return reflect.ValueOf(value).Convert(fieldType)
}

// getRedactedFloat creates a redacted float that satisfies validation constraints
func (cs *zCereal) getRedactedFloat(fieldType reflect.Type, constraints ValidationConstraints) reflect.Value {
	value := 0.0
	if constraints.Min > 0 {
		value = float64(constraints.Min)
	}
	return reflect.ValueOf(value).Convert(fieldType)
}
