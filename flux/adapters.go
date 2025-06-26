package flux

import (
	"fmt"

	"zbz/cereal"
)

// parseBytes returns raw file content as []byte using cereal
func parseBytes(content []byte) (any, error) {
	// Use cereal raw provider for consistent handling across ZBZ
	config := cereal.DefaultConfig()
	config.DefaultFormat = "raw"
	contract := cereal.NewRawProvider(config)
	
	var result []byte
	if err := contract.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse bytes with cereal: %w", err)
	}
	return result, nil
}

// parseText returns file content as string using cereal
func parseText(content []byte) (any, error) {
	// Use cereal string provider for consistent handling across ZBZ
	config := cereal.DefaultConfig()
	config.DefaultFormat = "string"
	contract := cereal.NewStringProvider(config)
	
	var result string
	if err := contract.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse text with cereal: %w", err)
	}
	return result, nil
}

// parseJSON parses JSON content into type T using cereal
func parseJSON[T any](content []byte) (any, error) {
	// Use cereal JSON provider for consistent serialization across ZBZ
	config := cereal.DefaultConfig()
	config.DefaultFormat = "json"
	contract := cereal.NewJSONProvider(config)
	
	var result T
	if err := contract.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON with cereal: %w", err)
	}
	return result, nil
}

// parseYAML parses YAML content into type T using cereal
func parseYAML[T any](content []byte) (any, error) {
	// Use cereal YAML provider (when available) for consistent serialization across ZBZ
	// For now, falls back to standard library but architecture is ready for cereal YAML provider
	config := cereal.DefaultConfig()
	config.DefaultFormat = "yaml"
	
	// Note: This would use cereal YAML provider when implemented
	// contract, err := cerealyaml.NewYAMLProvider(config)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to create YAML provider: %w", err)
	// }
	// var result T
	// if err := contract.Unmarshal(content, &result); err != nil {
	//     return nil, fmt.Errorf("failed to parse YAML with cereal: %w", err)
	// }
	// return result, nil
	
	// Temporary fallback until cereal YAML provider is available
	return nil, fmt.Errorf("YAML parsing via cereal not yet implemented - waiting for cereal YAML provider")
}