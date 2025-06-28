package cereal

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// zYaml implements the Serializer interface for YAML format
type zYaml struct{}

// Marshal serializes data to YAML with scoping (always applied)
func (y *zYaml) Marshal(v any, permissions ...string) ([]byte, error) {
	// Check convention-based scope requirements first
	if err := cereal.checkConventionScopes(v, permissions); err != nil {
		return nil, fmt.Errorf("scope check failed: %w", err)
	}
	
	// Apply scoping first (with redaction instead of omission)
	filtered := cereal.filterForMarshal(v, permissions)
	
	// Validate the scoped/redacted struct to ensure redacted values don't break validation
	if err := Validate(filtered); err != nil {
		return nil, err
	}
	
	return yaml.Marshal(filtered)
}

// Unmarshal deserializes YAML data with optional scoping validation
func (y *zYaml) Unmarshal(data []byte, v any, permissions ...string) error {
	// First unmarshal normally
	err := yaml.Unmarshal(data, v)
	if err != nil {
		return err
	}

	// Always apply scoping validation (zeros out restricted fields)
	if err := cereal.validateUnmarshalPermissions(v, permissions); err != nil {
		return err
	}
	
	// Validate struct after unmarshaling and scoping
	return Validate(v)
}
