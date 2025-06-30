package cereal

import (
	"encoding/json"
	
	"zbz/catalog"
)

// zJSON implements the Serializer interface for JSON format
type zJSON struct{}

// Marshal serializes data to JSON with scoping (always applied)
func (j *zJSON) Marshal(v any, permissions ...string) ([]byte, error) {
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
	
	result, err := json.Marshal(filtered)
	return result, err
}

// Unmarshal deserializes JSON data with optional scoping validation
func (j *zJSON) Unmarshal(data []byte, v any, permissions ...string) error {
	// First unmarshal normally
	err := json.Unmarshal(data, v)
	
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
