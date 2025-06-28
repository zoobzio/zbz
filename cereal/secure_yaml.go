package cereal

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

// SecureYAML provides catalog-integrated YAML serialization with encryption and security
type SecureYAML struct {
	encryptionService *EncryptionService
}

// NewSecureYAML creates a new secure YAML serializer
func NewSecureYAML(orgMasterKey []byte) *SecureYAML {
	return &SecureYAML{
		encryptionService: NewEncryptionService(orgMasterKey),
	}
}

// MarshalWithContext serializes data to YAML with full security context
func (s *SecureYAML) MarshalWithContext(v any, ctx SecurityContext) ([]byte, error) {
	// 1. Run user-registered security actions
	if err := runSecurityActions(v, "marshal", ctx); err != nil {
		return nil, fmt.Errorf("security action failed: %w", err)
	}
	
	// 2. Create a copy to avoid mutating original
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	
	copyValue := reflect.New(value.Type()).Elem()
	copyValue.Set(value)
	
	// 3. Apply encryption
	if err := applyFieldEncryption(copyValue, ctx, s.encryptionService); err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}
	
	// 4. Apply redaction
	if err := applyScopeRedaction(copyValue, ctx.Permissions); err != nil {
		return nil, fmt.Errorf("scope redaction failed: %w", err)
	}
	
	// 5. Validate
	if err := Validate(copyValue.Interface()); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// 6. Serialize to YAML
	return yaml.Marshal(copyValue.Interface())
}

// UnmarshalWithContext deserializes YAML data with security context
func (s *SecureYAML) UnmarshalWithContext(data []byte, v any, ctx SecurityContext) error {
	// 1. Run security actions
	if err := runSecurityActions(v, "unmarshal", ctx); err != nil {
		return fmt.Errorf("security action failed: %w", err)
	}
	
	// 2. Unmarshal YAML
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("yaml unmarshal failed: %w", err)
	}
	
	// 3. Decrypt
	value := reflect.ValueOf(v)
	if err := applyFieldDecryption(value, ctx, s.encryptionService); err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}
	
	// 4. Scope validation
	if err := applyScopeValidation(value, ctx.Permissions); err != nil {
		return fmt.Errorf("scope validation failed: %w", err)
	}
	
	// 5. Validate
	return Validate(v)
}

// Marshal provides backward compatibility
func (s *SecureYAML) Marshal(v any, permissions ...string) ([]byte, error) {
	ctx := SecurityContext{Permissions: permissions}
	return s.MarshalWithContext(v, ctx)
}

// Unmarshal provides backward compatibility
func (s *SecureYAML) Unmarshal(data []byte, v any, permissions ...string) error {
	ctx := SecurityContext{Permissions: permissions}
	return s.UnmarshalWithContext(data, v, ctx)
}

// Global secure YAML serializer
var SecureYAML_DefaultKey = NewSecureYAML([]byte("default-org-key-change-me"))

// Package-level convenience functions
func MarshalSecureYAML(v any, ctx SecurityContext) ([]byte, error) {
	return SecureYAML_DefaultKey.MarshalWithContext(v, ctx)
}

func UnmarshalSecureYAML(data []byte, v any, ctx SecurityContext) error {
	return SecureYAML_DefaultKey.UnmarshalWithContext(data, v, ctx)
}