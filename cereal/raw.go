package cereal

import (
	"fmt"
	"reflect"
)

// RawProvider implements CerealProvider for raw byte data (pass-through)
type RawProvider struct {
	config CerealConfig
}

// NewRawProvider creates a new raw bytes provider contract
func NewRawProvider(config CerealConfig) *CerealContract[[]byte] {
	// Apply defaults
	if config.Name == "" {
		config.Name = "raw"
	}
	
	provider := &RawProvider{
		config: config,
	}
	
	// Native type is just []byte
	var nativeBytes []byte
	
	return NewContract("raw", provider, nativeBytes, config)
}

// Marshal for raw provider just returns the bytes if input is []byte
func (r *RawProvider) Marshal(data any) ([]byte, error) {
	switch v := data.(type) {
	case []byte:
		// Make a copy to avoid mutations
		result := make([]byte, len(v))
		copy(result, v)
		return result, nil
	case string:
		// Convert string to bytes
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("raw provider can only marshal []byte or string, got %T", data)
	}
}

// Unmarshal for raw provider copies bytes to target
func (r *RawProvider) Unmarshal(data []byte, target any) error {
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	
	targetElem := targetVal.Elem()
	
	switch targetElem.Kind() {
	case reflect.Slice:
		// Check if it's []byte
		if targetElem.Type().Elem().Kind() == reflect.Uint8 {
			// Create new slice and copy data
			newSlice := make([]byte, len(data))
			copy(newSlice, data)
			targetElem.Set(reflect.ValueOf(newSlice))
			return nil
		}
		return fmt.Errorf("raw provider can only unmarshal to []byte, got %s", targetElem.Type())
	case reflect.String:
		// Convert bytes to string
		targetElem.SetString(string(data))
		return nil
	default:
		return fmt.Errorf("raw provider can only unmarshal to []byte or string, got %s", targetElem.Type())
	}
}

// MarshalScoped for raw data (no scoping - raw bytes have no fields)
func (r *RawProvider) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	// Raw bytes have no fields to scope, so this is identical to Marshal
	return r.Marshal(data)
}

// UnmarshalScoped for raw data (no scoping - raw bytes have no fields)
func (r *RawProvider) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	// Raw bytes have no fields to scope, so this is identical to Unmarshal
	return r.Unmarshal(data, target)
}

// ContentType returns the MIME type for binary data
func (r *RawProvider) ContentType() string {
	return "application/octet-stream"
}

// Format returns the format identifier
func (r *RawProvider) Format() string {
	return "raw"
}

// SupportsBinaryData returns true (this is specifically for binary data)
func (r *RawProvider) SupportsBinaryData() bool {
	return true
}

// SupportsStreaming returns false (not implemented yet)
func (r *RawProvider) SupportsStreaming() bool {
	return false
}

// Close cleans up the provider (raw provider has no cleanup needed)
func (r *RawProvider) Close() error {
	return nil
}