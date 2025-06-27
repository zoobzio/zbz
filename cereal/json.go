package cereal

import (
	"encoding/json"
)

// zJSON implements the Serializer interface for JSON format
type zJSON struct{}

// Marshal serializes data to JSON with scoping (always applied)
func (j *zJSON) Marshal(v any, permissions ...string) ([]byte, error) {
	// For marshal, completely omit filtered fields from output
	filtered := cereal.filterForMarshal(v, permissions)
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
	return cereal.validateUnmarshalPermissions(v, permissions)
}
