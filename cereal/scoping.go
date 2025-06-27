package cereal

import (
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

// filterForMarshal applies scoping for marshal operations by completely omitting filtered fields
func (cs *zCereal) filterForMarshal(data any, userPermissions []string) any {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return data // Return as-is for non-struct types
	}

	t := val.Type()
	fieldScopes := cs.parseFieldScopes(t)

	// If no scope restrictions exist, return original data
	if len(fieldScopes) == 0 {
		return data
	}

	// Create a map to hold filtered data - this ensures fields are truly omitted from output
	result := make(map[string]any)

	for i := range val.NumField() {
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
		if requiredPerms, hasScope := fieldScopes[fieldName]; hasScope {
			// Field has scope restrictions - check if user has access
			if cs.hasPermission(requiredPerms, userPermissions) {
				// User has access, include the field
				result[fieldName] = fieldValue.Interface()
			}
			// If no permission, field is completely omitted from result
		} else {
			// No scope restrictions - always include (includes fields with no scope tag)
			result[fieldName] = fieldValue.Interface()
		}
	}

	return result
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
