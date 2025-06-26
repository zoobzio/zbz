package yaml

import (
	"fmt"
	
	"gopkg.in/yaml.v3"
	
	"zbz/cereal"
)

// yamlProvider implements CerealProvider using YAML
type yamlProvider struct {
	config cereal.CerealConfig
	cereal *cereal.ZCereal // Reference to service for scoped operations (would be unexported in real impl)
}

// NewYAMLProvider creates a new YAML provider contract
func NewYAMLProvider(config cereal.CerealConfig) (*cereal.CerealContract[*yaml.Encoder], error) {
	// Apply defaults
	if config.Name == "" {
		config.Name = "yaml"
	}
	
	provider := &yamlProvider{
		config: config,
	}
	
	// Create a dummy encoder for the native type
	encoder := yaml.NewEncoder(nil)
	
	return cereal.NewContract("yaml", provider, encoder, config), nil
}

// Marshal serializes data to YAML
func (y *yamlProvider) Marshal(data any) ([]byte, error) {
	return yaml.Marshal(data)
}

// Unmarshal deserializes YAML data
func (y *yamlProvider) Unmarshal(data []byte, target any) error {
	return yaml.Unmarshal(data, target)
}

// MarshalScoped serializes data with field-level scope filtering (YAML doesn't typically need scoping)
func (y *yamlProvider) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	// For configuration files, scoping is usually not needed
	// But if cereal service is available, we can still filter
	if y.cereal != nil {
		filteredData := y.cereal.FilterFieldsForRead(data, userPermissions)
		return yaml.Marshal(filteredData)
	}
	
	// Fall back to regular marshal
	return y.Marshal(data)
}

// UnmarshalScoped deserializes YAML data with field-level scope validation
func (y *yamlProvider) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	// For configuration files, write scoping is usually not needed
	// But if cereal service is available, we can still validate
	if y.cereal != nil {
		// First, unmarshal into a temporary map to see what fields are being set
		var inputFields map[string]any
		if err := yaml.Unmarshal(data, &inputFields); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		// Validate write permissions for the fields being modified
		if err := y.cereal.ValidateWritePermissions(inputFields, target, userPermissions, operation); err != nil {
			return err
		}
	}
	
	// Perform the actual deserialization
	return yaml.Unmarshal(data, target)
}

// ContentType returns the MIME type for YAML
func (y *yamlProvider) ContentType() string {
	return "application/x-yaml"
}

// Format returns the format identifier
func (y *yamlProvider) Format() string {
	return "yaml"
}

// SupportsBinaryData returns false (YAML is text-based)
func (y *yamlProvider) SupportsBinaryData() bool {
	return false
}

// SupportsStreaming returns false (not implemented yet)
func (y *yamlProvider) SupportsStreaming() bool {
	return false
}

// Close cleans up the provider (YAML provider has no cleanup needed)
func (y *yamlProvider) Close() error {
	return nil
}

// MarshalIndent provides pretty-printed YAML (YAML is always indented)
func (y *yamlProvider) MarshalIndent(data any) ([]byte, error) {
	return y.Marshal(data) // YAML is always pretty-printed
}