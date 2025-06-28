package cereal

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// Test structs for merge testing
type UserConfig struct {
	Name     string            `merge:"replace"`
	Email    string            `merge:"replace"`
	Settings map[string]string `merge:"deep"`
	Features []string          `merge:"union"`
	Debug    bool              `merge:"replace"`
	Metadata *Metadata         `merge:"deep"`
}

type Metadata struct {
	Version   string    `merge:"replace"`
	CreatedAt time.Time `merge:"skip"`
	Tags      []string  `merge:"append"`
}

type DatabaseConfig struct {
	Host            string        `merge:"replace"`
	Port            int           `merge:"replace"`
	MaxConnections  int           `merge:"replace"`
	Timeout         time.Duration `merge:"replace"`
	Features        []string      `merge:"union"`
	ConnectionPool  map[string]int `merge:"deep"`
}

func TestBasicMerge(t *testing.T) {
	defaults := UserConfig{
		Name:  "Default User",
		Email: "default@example.com",
		Settings: map[string]string{
			"theme": "dark",
			"lang":  "en",
		},
		Features: []string{"basic", "auth"},
		Debug:    false,
	}

	overrides := UserConfig{
		Name:  "John Doe",
		Email: "", // Zero value should keep default
		Settings: map[string]string{
			"theme":  "light", // Override existing
			"locale": "us",    // Add new
		},
		Features: []string{"basic", "premium"}, // Union with defaults
		Debug:    true,
	}

	merged := Merge(defaults, overrides)

	// Verify basic field replacement
	if merged.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", merged.Name)
	}

	// Verify zero value handling
	if merged.Email != "default@example.com" {
		t.Errorf("Expected Email 'default@example.com', got '%s'", merged.Email)
	}

	// Verify map deep merge
	expectedSettings := map[string]string{
		"theme":  "light", // Overridden
		"lang":   "en",    // From defaults
		"locale": "us",    // Added from overrides
	}
	if !reflect.DeepEqual(merged.Settings, expectedSettings) {
		t.Errorf("Expected Settings %v, got %v", expectedSettings, merged.Settings)
	}

	// Verify array union
	expectedFeatures := []string{"basic", "auth", "premium"}
	if !reflect.DeepEqual(merged.Features, expectedFeatures) {
		t.Errorf("Expected Features %v, got %v", expectedFeatures, merged.Features)
	}

	// Verify boolean override
	if merged.Debug != true {
		t.Errorf("Expected Debug true, got %v", merged.Debug)
	}
}

func TestStructTagMerging(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	defaults := UserConfig{
		Name: "Default",
		Metadata: &Metadata{
			Version:   "1.0",
			CreatedAt: now,
			Tags:      []string{"default", "system"},
		},
	}

	overrides := UserConfig{
		Name: "Override",
		Metadata: &Metadata{
			Version:   "2.0",
			CreatedAt: later, // Should be skipped
			Tags:      []string{"user", "custom"},
		},
	}

	merged := Merge(defaults, overrides)

	// Verify replace tag on Name
	if merged.Name != "Override" {
		t.Errorf("Expected Name 'Override', got '%s'", merged.Name)
	}

	// Verify deep merge on Metadata
	if merged.Metadata == nil {
		t.Fatal("Expected Metadata to be merged, got nil")
	}

	// Verify replace tag on Version
	if merged.Metadata.Version != "2.0" {
		t.Errorf("Expected Version '2.0', got '%s'", merged.Metadata.Version)
	}

	// Verify skip tag on CreatedAt
	if !merged.Metadata.CreatedAt.Equal(now) {
		t.Errorf("Expected CreatedAt to be skipped and remain %v, got %v", now, merged.Metadata.CreatedAt)
	}

	// Verify append tag on Tags
	expectedTags := []string{"default", "system", "user", "custom"}
	if !reflect.DeepEqual(merged.Metadata.Tags, expectedTags) {
		t.Errorf("Expected Tags %v, got %v", expectedTags, merged.Metadata.Tags)
	}
}

func TestArrayMergeStrategies(t *testing.T) {
	// Create structs without merge tags to test pure strategy behavior
	type SimpleConfig struct {
		Features []string
	}

	defaults := SimpleConfig{
		Features: []string{"ssl", "pooling"},
	}

	overrides := SimpleConfig{
		Features: []string{"ssl", "monitoring"},
	}

	// Test ArrayReplace strategy
	options := MergeOptions{
		ArrayStrategy:  ArrayReplace,
		StructStrategy: StructDeepMerge,
		MapStrategy:    MapDeepMerge,
		NilStrategy:    NilKeepDefaults,
		FieldRules:     make(map[string]FieldMergeRule),
	}

	merged := MergeWithOptions(defaults, overrides, options)
	expectedReplace := []string{"ssl", "monitoring"}
	if !reflect.DeepEqual(merged.Features, expectedReplace) {
		t.Errorf("ArrayReplace: Expected %v, got %v", expectedReplace, merged.Features)
	}

	// Test ArrayAppend strategy
	options.ArrayStrategy = ArrayAppend
	merged = MergeWithOptions(defaults, overrides, options)
	expectedAppend := []string{"ssl", "pooling", "ssl", "monitoring"}
	if !reflect.DeepEqual(merged.Features, expectedAppend) {
		t.Errorf("ArrayAppend: Expected %v, got %v", expectedAppend, merged.Features)
	}

	// Test ArrayUnion strategy
	options.ArrayStrategy = ArrayUnion
	merged = MergeWithOptions(defaults, overrides, options)
	expectedUnion := []string{"ssl", "pooling", "monitoring"}
	if !reflect.DeepEqual(merged.Features, expectedUnion) {
		t.Errorf("ArrayUnion: Expected %v, got %v", expectedUnion, merged.Features)
	}
}

func TestMapMergeStrategies(t *testing.T) {
	defaults := DatabaseConfig{
		ConnectionPool: map[string]int{
			"read":  5,
			"write": 2,
		},
	}

	overrides := DatabaseConfig{
		ConnectionPool: map[string]int{
			"write":    10, // Override existing
			"archive":  1,  // Add new
		},
	}

	// Test MapDeepMerge (default)
	merged := Merge(defaults, overrides)
	expected := map[string]int{
		"read":    5,  // From defaults
		"write":   10, // Overridden
		"archive": 1,  // Added
	}
	if !reflect.DeepEqual(merged.ConnectionPool, expected) {
		t.Errorf("MapDeepMerge: Expected %v, got %v", expected, merged.ConnectionPool)
	}

	// Test MapReplace
	options := MergeOptions{
		ArrayStrategy:  ArrayReplace,
		StructStrategy: StructDeepMerge,
		MapStrategy:    MapReplace,
		NilStrategy:    NilKeepDefaults,
		FieldRules:     make(map[string]FieldMergeRule),
	}

	merged = MergeWithOptions(defaults, overrides, options)
	expectedReplace := map[string]int{
		"write":   10,
		"archive": 1,
	}
	if !reflect.DeepEqual(merged.ConnectionPool, expectedReplace) {
		t.Errorf("MapReplace: Expected %v, got %v", expectedReplace, merged.ConnectionPool)
	}
}

func TestNilHandling(t *testing.T) {
	defaults := UserConfig{
		Name:     "Default",
		Email:    "default@example.com",
		Metadata: &Metadata{Version: "1.0"},
	}

	overrides := UserConfig{
		Name:     "", // Zero value
		Email:    "override@example.com",
		Metadata: nil, // Nil pointer
	}

	// Test NilKeepDefaults (default behavior)
	merged := Merge(defaults, overrides)

	if merged.Name != "Default" {
		t.Errorf("Expected Name to keep default 'Default', got '%s'", merged.Name)
	}

	if merged.Email != "override@example.com" {
		t.Errorf("Expected Email to be overridden, got '%s'", merged.Email)
	}

	if merged.Metadata == nil || merged.Metadata.Version != "1.0" {
		t.Errorf("Expected Metadata to keep default, got %v", merged.Metadata)
	}

	// Test NilAllowOverride
	options := MergeOptions{
		ArrayStrategy:  ArrayReplace,
		StructStrategy: StructDeepMerge,
		MapStrategy:    MapDeepMerge,
		NilStrategy:    NilAllowOverride,
		FieldRules:     make(map[string]FieldMergeRule),
	}

	merged = MergeWithOptions(defaults, overrides, options)

	if merged.Name != "" {
		t.Errorf("NilAllowOverride: Expected Name to be empty, got '%s'", merged.Name)
	}

	if merged.Metadata != nil {
		t.Errorf("NilAllowOverride: Expected Metadata to be nil, got %v", merged.Metadata)
	}
}

func TestPointerMerging(t *testing.T) {
	defaults := UserConfig{
		Metadata: &Metadata{
			Version: "1.0",
			Tags:    []string{"default"},
		},
	}

	overrides := UserConfig{
		Metadata: &Metadata{
			Version: "2.0",
			Tags:    []string{"override"},
		},
	}

	merged := Merge(defaults, overrides)

	if merged.Metadata == nil {
		t.Fatal("Expected Metadata to be merged, got nil")
	}

	if merged.Metadata.Version != "2.0" {
		t.Errorf("Expected Version '2.0', got '%s'", merged.Metadata.Version)
	}

	// Verify it's a new pointer, not the same as input
	if merged.Metadata == defaults.Metadata || merged.Metadata == overrides.Metadata {
		t.Error("Expected new pointer for merged Metadata")
	}
}

func TestMergeMultiple(t *testing.T) {
	config1 := UserConfig{
		Name:     "Config1",
		Features: []string{"feature1"},
		Settings: map[string]string{"key1": "value1"},
	}

	config2 := UserConfig{
		Email:    "config2@example.com",
		Features: []string{"feature2"},
		Settings: map[string]string{"key2": "value2"},
	}

	config3 := UserConfig{
		Debug:    true,
		Features: []string{"feature3"},
		Settings: map[string]string{"key3": "value3"},
	}

	merged := MergeMultiple(config1, config2, config3)

	// Last non-zero value should win for simple fields
	if merged.Name != "Config1" {
		t.Errorf("Expected Name 'Config1', got '%s'", merged.Name)
	}
	if merged.Email != "config2@example.com" {
		t.Errorf("Expected Email 'config2@example.com', got '%s'", merged.Email)
	}
	if merged.Debug != true {
		t.Errorf("Expected Debug true, got %v", merged.Debug)
	}

	// Arrays should union all values
	expectedFeatures := []string{"feature1", "feature2", "feature3"}
	if !reflect.DeepEqual(merged.Features, expectedFeatures) {
		t.Errorf("Expected Features %v, got %v", expectedFeatures, merged.Features)
	}

	// Maps should merge all values
	expectedSettings := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	if !reflect.DeepEqual(merged.Settings, expectedSettings) {
		t.Errorf("Expected Settings %v, got %v", expectedSettings, merged.Settings)
	}
}

func TestMergeWithValidation(t *testing.T) {
	defaults := DatabaseConfig{
		Host:           "localhost",
		Port:           5432,
		MaxConnections: 10,
		Timeout:        30 * time.Second,
	}

	validOverrides := DatabaseConfig{
		MaxConnections: 50,
	}

	invalidOverrides := DatabaseConfig{
		MaxConnections: -5, // Invalid value
	}

	validator := func(config DatabaseConfig) error {
		if config.MaxConnections <= 0 {
			return fmt.Errorf("max_connections must be positive, got %d", config.MaxConnections)
		}
		return nil
	}

	// Test valid merge
	merged, err := MergeWithValidation(defaults, validOverrides, validator)
	if err != nil {
		t.Errorf("Expected valid merge to succeed, got error: %v", err)
	}
	if merged.MaxConnections != 50 {
		t.Errorf("Expected MaxConnections 50, got %d", merged.MaxConnections)
	}

	// Test invalid merge - should return defaults
	merged, err = MergeWithValidation(defaults, invalidOverrides, validator)
	if err == nil {
		t.Error("Expected invalid merge to fail")
	}
	if merged.MaxConnections != 10 {
		t.Errorf("Expected MaxConnections to revert to default 10, got %d", merged.MaxConnections)
	}
}

func TestEmptyMergeMultiple(t *testing.T) {
	// Test empty slice
	var result UserConfig = MergeMultiple[UserConfig]()
	if result.Name != "" {
		t.Errorf("Expected empty result for no sources, got %+v", result)
	}

	// Test single source
	single := UserConfig{Name: "Single"}
	result = MergeMultiple(single)
	if result.Name != "Single" {
		t.Errorf("Expected single source to be returned unchanged, got %+v", result)
	}
}

func TestComplexNestedMerging(t *testing.T) {
	type NestedConfig struct {
		Level1 struct {
			Level2 struct {
				Value string            `merge:"replace"`
				Map   map[string]string `merge:"deep"`
			} `merge:"deep"`
			Array []string `merge:"union"`
		} `merge:"deep"`
	}

	defaults := NestedConfig{}
	defaults.Level1.Level2.Value = "default"
	defaults.Level1.Level2.Map = map[string]string{"key1": "value1"}
	defaults.Level1.Array = []string{"item1"}

	overrides := NestedConfig{}
	overrides.Level1.Level2.Value = "override"
	overrides.Level1.Level2.Map = map[string]string{"key2": "value2"}
	overrides.Level1.Array = []string{"item2"}

	merged := Merge(defaults, overrides)

	if merged.Level1.Level2.Value != "override" {
		t.Errorf("Expected nested Value 'override', got '%s'", merged.Level1.Level2.Value)
	}

	expectedMap := map[string]string{"key1": "value1", "key2": "value2"}
	if !reflect.DeepEqual(merged.Level1.Level2.Map, expectedMap) {
		t.Errorf("Expected nested Map %v, got %v", expectedMap, merged.Level1.Level2.Map)
	}

	expectedArray := []string{"item1", "item2"}
	if !reflect.DeepEqual(merged.Level1.Array, expectedArray) {
		t.Errorf("Expected nested Array %v, got %v", expectedArray, merged.Level1.Array)
	}
}