package cereal

import (
	"fmt"
	"reflect"

	"github.com/pelletier/go-toml/v2"
)

// SecureTOML provides catalog-integrated TOML serialization with encryption and security
type SecureTOML struct {
	encryptionService *EncryptionService
}

// NewSecureTOML creates a new secure TOML serializer
func NewSecureTOML(orgMasterKey []byte) *SecureTOML {
	return &SecureTOML{
		encryptionService: NewEncryptionService(orgMasterKey),
	}
}

// MarshalWithContext serializes data to TOML with full security context
func (s *SecureTOML) MarshalWithContext(v any, ctx SecurityContext) ([]byte, error) {
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
	
	// 6. Serialize to TOML
	return toml.Marshal(copyValue.Interface())
}

// UnmarshalWithContext deserializes TOML data with security context
func (s *SecureTOML) UnmarshalWithContext(data []byte, v any, ctx SecurityContext) error {
	// 1. Run security actions
	if err := runSecurityActions(v, "unmarshal", ctx); err != nil {
		return fmt.Errorf("security action failed: %w", err)
	}
	
	// 2. Unmarshal TOML
	if err := toml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("toml unmarshal failed: %w", err)
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
func (s *SecureTOML) Marshal(v any, permissions ...string) ([]byte, error) {
	ctx := SecurityContext{Permissions: permissions}
	return s.MarshalWithContext(v, ctx)
}

// Unmarshal provides backward compatibility
func (s *SecureTOML) Unmarshal(data []byte, v any, permissions ...string) error {
	ctx := SecurityContext{Permissions: permissions}
	return s.UnmarshalWithContext(data, v, ctx)
}

// Global secure TOML serializer
var SecureTOML_DefaultKey = NewSecureTOML([]byte("default-org-key-change-me"))

// Package-level convenience functions
func MarshalSecureTOML(v any, ctx SecurityContext) ([]byte, error) {
	return SecureTOML_DefaultKey.MarshalWithContext(v, ctx)
}

func UnmarshalSecureTOML(data []byte, v any, ctx SecurityContext) error {
	return SecureTOML_DefaultKey.UnmarshalWithContext(data, v, ctx)
}