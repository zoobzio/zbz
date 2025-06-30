package cereal

import (
	"github.com/pelletier/go-toml/v2"
	
	"zbz/catalog"
)

// zTOML implements the Serializer interface for TOML format
type zTOML struct{}

// Marshal serializes data to TOML with scoping (always applied)
func (t *zTOML) Marshal(v any, permissions ...string) ([]byte, error) {
	// Use catalog-based scoping instead of reflection
	filtered := catalogScoper.FilterForMarshal(v, permissions)
	
	// Emit marshal event for monitoring/auditing
	var err error
	defer func() {
		// Get model type for event
		modelType := "unknown"
		if metadata := catalog.ExtractAndCacheMetadata(v); metadata.TypeName != "" {
			modelType = metadata.TypeName
		}
		emitMarshalEvent(modelType, permissions, err == nil, err)
	}()
	
	// Validate the scoped/redacted data to ensure redacted values don't break validation
	if err = Validate(filtered); err != nil {
		return nil, err
	}
	
	result, err := toml.Marshal(filtered)
	return result, err
}

// Unmarshal deserializes TOML data with optional scoping validation
func (t *zTOML) Unmarshal(data []byte, v any, permissions ...string) error {
	// First unmarshal normally
	err := toml.Unmarshal(data, v)
	
	// Emit unmarshal event for monitoring/auditing
	defer func() {
		// Get model type for event
		modelType := "unknown"
		if metadata := catalog.ExtractAndCacheMetadata(v); metadata.TypeName != "" {
			modelType = metadata.TypeName
		}
		emitUnmarshalEvent(modelType, permissions, err == nil, err)
	}()
	
	if err != nil {
		return err
	}

	// Always apply scoping validation using catalog-based system
	if err = catalogScoper.ValidateUnmarshalPermissions(v, permissions); err != nil {
		return err
	}
	
	// Validate struct after unmarshaling and scoping
	err = Validate(v)
	return err
}