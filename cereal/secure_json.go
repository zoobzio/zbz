package cereal

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// SecureJSON provides catalog-integrated JSON serialization with encryption and security
type SecureJSON struct {
	encryptionService *EncryptionService
}

// NewSecureJSON creates a new secure JSON serializer
func NewSecureJSON(orgMasterKey []byte) *SecureJSON {
	return &SecureJSON{
		encryptionService: NewEncryptionService(orgMasterKey),
	}
}

// MarshalWithContext serializes data with full security context
func (s *SecureJSON) MarshalWithContext(v any, ctx SecurityContext) ([]byte, error) {
	// 1. Run user-registered security actions (can break the chain)
	if err := runSecurityActions(v, "marshal", ctx); err != nil {
		return nil, fmt.Errorf("security action failed: %w", err)
	}
	
	// 2. Create a copy of the data to avoid mutating the original
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	
	// Create a new instance of the same type
	copyValue := reflect.New(value.Type()).Elem()
	copyValue.Set(value)
	
	// 3. Apply field-level encryption based on catalog metadata
	if err := applyFieldEncryption(copyValue, ctx, s.encryptionService); err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}
	
	// 4. Apply scope-based redaction
	if err := applyScopeRedaction(copyValue, ctx.Permissions); err != nil {
		return nil, fmt.Errorf("scope redaction failed: %w", err)
	}
	
	// 5. Validate the processed struct
	if err := Validate(copyValue.Interface()); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// 6. Serialize to JSON
	return json.Marshal(copyValue.Interface())
}

// UnmarshalWithContext deserializes JSON data with security context
func (s *SecureJSON) UnmarshalWithContext(data []byte, v any, ctx SecurityContext) error {
	// 1. Run security actions for unmarshal operation
	if err := runSecurityActions(v, "unmarshal", ctx); err != nil {
		return fmt.Errorf("security action failed: %w", err)
	}
	
	// 2. Unmarshal JSON data
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}
	
	// 3. Apply field-level decryption
	value := reflect.ValueOf(v)
	if err := applyFieldDecryption(value, ctx, s.encryptionService); err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}
	
	// 4. Validate against scoping (zero out restricted fields if user lacks permission)
	if err := applyScopeValidation(value, ctx.Permissions); err != nil {
		return fmt.Errorf("scope validation failed: %w", err)
	}
	
	// 5. Final validation
	return Validate(v)
}

// Marshal provides backward compatibility with simple permission list
func (s *SecureJSON) Marshal(v any, permissions ...string) ([]byte, error) {
	ctx := SecurityContext{
		Permissions: permissions,
	}
	return s.MarshalWithContext(v, ctx)
}

// Unmarshal provides backward compatibility with simple permission list
func (s *SecureJSON) Unmarshal(data []byte, v any, permissions ...string) error {
	ctx := SecurityContext{
		Permissions: permissions,
	}
	return s.UnmarshalWithContext(data, v, ctx)
}

// applyScopeValidation zeros out fields that user doesn't have permission to see
func applyScopeValidation(value reflect.Value, permissions []string) error {
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	
	if value.Kind() != reflect.Struct {
		return nil
	}
	
	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		fieldValue := value.Field(i)
		
		if !fieldValue.CanSet() {
			continue
		}
		
		// Check scope requirements
		scopeTag := field.Tag.Get("scope")
		if scopeTag == "" {
			continue // No scope restriction
		}
		
		// Check if user has required scope
		hasScope := false
		for _, userScope := range permissions {
			if userScope == scopeTag {
				hasScope = true
				break
			}
		}
		
		if !hasScope {
			// Zero out the field if user lacks permission
			fieldValue.Set(reflect.Zero(fieldValue.Type()))
		}
	}
	
	return nil
}

// Global secure serializers with default empty org key
var (
	SecureJSON_DefaultKey = NewSecureJSON([]byte("default-org-key-change-me"))
)

// Package-level convenience functions using default serializer
func MarshalSecure(v any, ctx SecurityContext) ([]byte, error) {
	return SecureJSON_DefaultKey.MarshalWithContext(v, ctx)
}

func UnmarshalSecure(data []byte, v any, ctx SecurityContext) error {
	return SecureJSON_DefaultKey.UnmarshalWithContext(data, v, ctx)
}