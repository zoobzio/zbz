package flux

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// parseBytes returns raw file content as []byte
func parseBytes(content []byte) (any, error) {
	return content, nil
}

// parseText returns file content as string
func parseText(content []byte) (any, error) {
	return string(content), nil
}

// parseJSON parses JSON content into type T
func parseJSON[T any](content []byte) (any, error) {
	var result T
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return result, nil
}

// parseYAML parses YAML content into type T
func parseYAML[T any](content []byte) (any, error) {
	var result T
	if err := yaml.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return result, nil
}