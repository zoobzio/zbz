package zbz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Cereal handles field-level scoped serialization and deserialization
type Cereal interface {
	// Serialization with scope filtering
	SerializeJSON(data any, userPermissions []string) ([]byte, error)
	SerializeProtobuf(data any, userPermissions []string) ([]byte, error)
	
	// Deserialization with scope validation for updates
	DeserializeJSON(jsonData []byte, target any, userPermissions []string, operation string) error
	
	// Field access validation
	ValidateFieldAccess(fieldName string, userPermissions []string, operation string) error
	GetFieldScopes(target any) map[string][]string
}

// SerializationFormat represents supported serialization formats
type SerializationFormat string

const (
	FormatJSON     SerializationFormat = "json"
	FormatProtobuf SerializationFormat = "protobuf"
)

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

// zCereal implements the Cereal interface
type zCereal struct{}

// NewCereal creates a new Cereal instance
func NewCereal() Cereal {
	return &zCereal{}
}

// parseFieldScopes extracts scope information from struct tags
func (c *zCereal) parseFieldScopes(field reflect.StructField) FieldScope {
	scopeTag := field.Tag.Get("scope")
	if scopeTag == "" {
		return FieldScope{}
	}

	scope := FieldScope{}
	
	// Parse scope tag format: "read:users:email,write:admin:users"
	parts := strings.Split(scopeTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Determine if this is read or write permission
		if strings.HasPrefix(part, "read:") || (!strings.Contains(part, ":write") && !strings.Contains(part, ":create") && !strings.Contains(part, ":update")) {
			// Default to read permission or explicit read permission
			perm := strings.TrimPrefix(part, "read:")
			scope.Read = append(scope.Read, perm)
		} else if strings.Contains(part, ":write") || strings.Contains(part, ":create") || strings.Contains(part, ":update") {
			// Write/create/update permission
			scope.Write = append(scope.Write, part)
		}
	}

	return scope
}

// GetFieldScopes returns a map of field names to their scope requirements
func (c *zCereal) GetFieldScopes(target any) map[string][]string {
	result := make(map[string][]string)
	
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	if val.Kind() != reflect.Struct {
		return result
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		
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

		// Parse scopes
		fieldScope := c.parseFieldScopes(field)
		if len(fieldScope.Read) > 0 || len(fieldScope.Write) > 0 {
			allScopes := append(fieldScope.Read, fieldScope.Write...)
			result[fieldName] = allScopes
		}
	}

	return result
}

// hasPermission checks if user has any of the required permissions
func (c *zCereal) hasPermission(userPermissions []string, requiredPermissions []string) bool {
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

// ValidateFieldAccess validates if user can access a field for the given operation
func (c *zCereal) ValidateFieldAccess(fieldName string, userPermissions []string, operation string) error {
	// For now, this is a simple implementation
	// In a real implementation, you'd look up the field's scope requirements
	return nil
}

// SerializeJSON serializes data to JSON with field-level scope filtering
func (c *zCereal) SerializeJSON(data any, userPermissions []string) ([]byte, error) {
	filteredData := c.filterFieldsForRead(data, userPermissions)
	return json.Marshal(filteredData)
}

// SerializeProtobuf serializes data to Protobuf with field-level scope filtering
func (c *zCereal) SerializeProtobuf(data any, userPermissions []string) ([]byte, error) {
	// For now, protobuf support is placeholder
	// In a real implementation, you'd integrate with protobuf reflection
	return nil, fmt.Errorf("protobuf serialization not yet implemented")
}

// filterFieldsForRead removes fields that the user doesn't have read access to
func (c *zCereal) filterFieldsForRead(data any, userPermissions []string) any {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return data // Return as-is for non-struct types
	}

	// Create a new struct with the same type
	newVal := reflect.New(val.Type()).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Parse field scopes
		fieldScope := c.parseFieldScopes(field)

		// Check read permission
		if c.hasPermission(userPermissions, fieldScope.Read) {
			// User has read access, copy the field
			if newVal.Field(i).CanSet() {
				newVal.Field(i).Set(fieldValue)
			}
		}
		// If no read permission, field remains zero value (effectively filtered out)
	}

	return newVal.Interface()
}

// DeserializeJSON deserializes JSON data with field-level scope validation for updates
func (c *zCereal) DeserializeJSON(jsonData []byte, target any, userPermissions []string, operation string) error {
	// First, unmarshal into a temporary map to see what fields are being set
	var inputFields map[string]any
	if err := json.Unmarshal(jsonData, &inputFields); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate that user has write access to all fields being updated
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct")
	}

	typ := val.Type()
	fieldMap := make(map[string]reflect.StructField)
	
	// Build field name mapping
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
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
			fieldScope := c.parseFieldScopes(structField)
			
			// For write operations, check write permissions
			var requiredPerms []string
			if operation == OperationCreate {
				// For create, allow if user has write OR create permission
				requiredPerms = fieldScope.Write
			} else if operation == OperationUpdate {
				// For update, require write permission
				requiredPerms = fieldScope.Write
			}

			if !c.hasPermission(userPermissions, requiredPerms) {
				unauthorized = append(unauthorized, fieldName)
			}
		}
	}

	// If any fields are unauthorized, return error (no silent failure)
	if len(unauthorized) > 0 {
		return fmt.Errorf("insufficient permissions to modify fields: %s", strings.Join(unauthorized, ", "))
	}

	// If all permissions check out, perform the actual deserialization
	return json.Unmarshal(jsonData, target)
}

// Global cereal instance
var cereal Cereal

// GetCereal returns the global cereal instance
func GetCereal() Cereal {
	return cereal
}

// SerializeWithScopes is a convenience function for scoped serialization
func SerializeWithScopes(ctx *gin.Context, data any, format SerializationFormat) ([]byte, error) {
	// Get user permissions from context
	permissions, exists := ctx.Get("permissions")
	if !exists {
		permissions = []string{} // No permissions
	}

	userPerms, ok := permissions.([]string)
	if !ok {
		userPerms = []string{}
	}

	switch format {
	case FormatJSON:
		return cereal.SerializeJSON(data, userPerms)
	case FormatProtobuf:
		return cereal.SerializeProtobuf(data, userPerms)
	default:
		return nil, fmt.Errorf("unsupported serialization format: %s", format)
	}
}

// DeserializeWithScopes is a convenience function for scoped deserialization
func DeserializeWithScopes(ctx *gin.Context, jsonData []byte, target any, operation string) error {
	// Get user permissions from context
	permissions, exists := ctx.Get("permissions")
	if !exists {
		return fmt.Errorf("no permissions found in context")
	}

	userPerms, ok := permissions.([]string)
	if !ok {
		return fmt.Errorf("invalid permissions type in context")
	}

	return cereal.DeserializeJSON(jsonData, target, userPerms, operation)
}

// RespondWithScopedJSON sends a JSON response with field-level scoping
func RespondWithScopedJSON(ctx *gin.Context, status int, data any) {
	jsonData, err := SerializeWithScopes(ctx, data, FormatJSON)
	if err != nil {
		Log.Error("Failed to serialize response with scopes", zap.Error(err))
		ctx.Set("error_message", "Failed to serialize response")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.Data(status, "application/json", jsonData)
}

// Initialize cereal instance
func init() {
	cereal = NewCereal()
}