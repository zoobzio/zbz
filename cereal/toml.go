package cereal

import (
	"github.com/pelletier/go-toml/v2"
)

// zTOML implements the Serializer interface for TOML format
type zTOML struct{}

// Marshal serializes data to TOML with scoping (always applied)
func (t *zTOML) Marshal(v any, permissions ...string) ([]byte, error) {
	// For marshal, completely omit filtered fields from output
	filtered := cereal.filterForMarshal(v, permissions)
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
	return cereal.validateUnmarshalPermissions(v, permissions)
}