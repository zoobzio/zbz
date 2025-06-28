package cereal

import (
	"fmt"
	"github.com/pelletier/go-toml/v2"
)

// zTOML implements the Serializer interface for TOML format
type zTOML struct{}

// Marshal serializes data to TOML with scoping (always applied)
func (t *zTOML) Marshal(v any, permissions ...string) ([]byte, error) {
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
	
	return toml.Marshal(filtered)
}

// Unmarshal deserializes TOML data with optional scoping validation
func (t *zTOML) Unmarshal(data []byte, v any, permissions ...string) error {
	// First unmarshal normally
	err := toml.Unmarshal(data, v)
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