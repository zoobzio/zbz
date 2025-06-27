package cereal

import (
	"gopkg.in/yaml.v3"
)

// zYaml implements the Serializer interface for YAML format
type zYaml struct{}

// Marshal serializes data to YAML with scoping (always applied)
func (y *zYaml) Marshal(v any, permissions ...string) ([]byte, error) {
	// For marshal, completely omit filtered fields from output
	filtered := cereal.filterForMarshal(v, permissions)
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
	return cereal.validateUnmarshalPermissions(v, permissions)
}
