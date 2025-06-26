package cereal

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// JSONProvider implements CerealProvider using the standard library JSON package
type JSONProvider struct {
	config CerealConfig
	cereal *zCereal // Reference to service for scoped operations
}

// NewJSONProvider creates a new JSON provider contract
func NewJSONProvider(config CerealConfig) *CerealContract[*json.Encoder] {
	// Apply defaults
	if config.Name == "" {
		config.Name = "json"
	}
	
	provider := &JSONProvider{
		config: config,
	}
	
	// Create a dummy encoder for the native type (users can create their own if needed)
	encoder := json.NewEncoder(nil)
	
	return NewContract("json", provider, encoder, config)
}

// setCereal sets the service reference (called by the service during initialization)
func (j *JSONProvider) setCereal(cereal *zCereal) {
	j.cereal = cereal
}

// Marshal serializes data to JSON
func (j *JSONProvider) Marshal(data any) ([]byte, error) {
	return json.Marshal(data)
}

// Unmarshal deserializes JSON data
func (j *JSONProvider) Unmarshal(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

// MarshalScoped serializes data with field-level scope filtering
func (j *JSONProvider) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	// If no cereal service reference, fall back to regular marshal
	if j.cereal == nil {
		return j.Marshal(data)
	}
	
	// Filter fields based on read permissions
	filteredData := j.cereal.filterFieldsForRead(data, userPermissions)
	return json.Marshal(filteredData)
}

// UnmarshalScoped deserializes JSON data with field-level scope validation
func (j *JSONProvider) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	// If no cereal service reference, fall back to regular unmarshal
	if j.cereal == nil {
		return j.Unmarshal(data, target)
	}
	
	// First, unmarshal into a temporary map to see what fields are being set
	var inputFields map[string]any
	if err := json.Unmarshal(data, &inputFields); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate write permissions for the fields being modified
	if err := j.cereal.validateWritePermissions(inputFields, target, userPermissions, operation); err != nil {
		return err
	}

	// If all permissions check out, perform the actual deserialization
	return json.Unmarshal(data, target)
}

// ContentType returns the MIME type for JSON
func (j *JSONProvider) ContentType() string {
	return "application/json"
}

// Format returns the format identifier
func (j *JSONProvider) Format() string {
	return "json"
}

// SupportsBinaryData returns false (JSON is text-based)
func (j *JSONProvider) SupportsBinaryData() bool {
	return false
}

// SupportsStreaming returns false (not implemented yet)
func (j *JSONProvider) SupportsStreaming() bool {
	return false
}

// Close cleans up the provider (JSON provider has no cleanup needed)
func (j *JSONProvider) Close() error {
	return nil
}

// MarshalIndent provides pretty-printed JSON for development/debugging
func (j *JSONProvider) MarshalIndent(data any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(data, prefix, indent)
}

// MarshalScopedIndent provides pretty-printed JSON with scoping
func (j *JSONProvider) MarshalScopedIndent(data any, userPermissions []string, prefix, indent string) ([]byte, error) {
	if j.cereal == nil {
		return j.MarshalIndent(data, prefix, indent)
	}
	
	filteredData := j.cereal.filterFieldsForRead(data, userPermissions)
	return json.MarshalIndent(filteredData, prefix, indent)
}