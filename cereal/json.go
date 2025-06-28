package cereal

import (
	"encoding/json"
	"fmt"
)

// zJSON implements the Serializer interface for JSON format
type zJSON struct{}

// Marshal serializes data to JSON with scoping (always applied)
func (j *zJSON) Marshal(v any, permissions ...string) ([]byte, error) {
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
	
	return json.Marshal(filtered)
}

// Unmarshal deserializes JSON data with optional scoping validation
func (j *zJSON) Unmarshal(data []byte, v any, permissions ...string) error {
	// First unmarshal normally
	err := json.Unmarshal(data, v)
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
