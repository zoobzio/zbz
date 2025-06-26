package cache

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// CerealSerializerManager replaces SerializerManager using the cereal service
type CerealSerializerManager struct {
	defaultFormat string
	cerealConfig  cereal.CerealConfig
}

// NewCerealSerializerManager creates a new cereal-based serializer manager
func NewCerealSerializerManager(defaultFormat string) *CerealSerializerManager {
	config := cereal.DefaultConfig()
	config.Name = "cache-serializer"
	config.DefaultFormat = defaultFormat
	config.EnableCaching = true  // Enable caching for performance
	config.EnableScoping = false // Cache doesn't need scoping by default

	return &CerealSerializerManager{
		defaultFormat: defaultFormat,
		cerealConfig:  config,
	}
}

// ForType returns a type-specific serializer wrapper that delegates to cereal
func (csm *CerealSerializerManager) ForType[T any]() Serializer[T] {
	// Determine appropriate provider based on type
	var contract interface{}
	
	switch any(*new(T)).(type) {
	case []byte:
		contract = cereal.NewRawProvider(csm.cerealConfig)
	case string:
		contract = cereal.NewStringProvider(csm.cerealConfig)
	default:
		// Struct types use JSON
		contract = cereal.NewJSONProvider(csm.cerealConfig)
	}
	
	return &CerealTypeSerializer[T]{
		contract: contract,
	}
}

// CerealTypeSerializer wraps cereal providers to match the cache Serializer[T] interface
type CerealTypeSerializer[T any] struct {
	contract interface{} // This will be a *cereal.CerealContract[NativeType]
}

// Marshal delegates to cereal
func (cts *CerealTypeSerializer[T]) Marshal(v T) ([]byte, error) {
	// Use the contract's provider
	switch c := cts.contract.(type) {
	case *cereal.CerealContract[*json.Encoder]:
		return c.Marshal(v)
	case *cereal.CerealContract[[]byte]:
		return c.Marshal(v)
	case *cereal.CerealContract[string]:
		return c.Marshal(v)
	default:
		return nil, fmt.Errorf("unsupported contract type: %T", c)
	}
}

// Unmarshal delegates to cereal
func (cts *CerealTypeSerializer[T]) Unmarshal(data []byte, v *T) error {
	// Use the contract's provider
	switch c := cts.contract.(type) {
	case *cereal.CerealContract[*json.Encoder]:
		return c.Unmarshal(data, v)
	case *cereal.CerealContract[[]byte]:
		return c.Unmarshal(data, v)
	case *cereal.CerealContract[string]:
		return c.Unmarshal(data, v)
	default:
		return fmt.Errorf("unsupported contract type: %T", c)
	}
}

// ContentType returns the MIME type from the provider
func (cts *CerealTypeSerializer[T]) ContentType() string {
	switch c := cts.contract.(type) {
	case *cereal.CerealContract[*json.Encoder]:
		return c.Provider().ContentType()
	case *cereal.CerealContract[[]byte]:
		return c.Provider().ContentType()
	case *cereal.CerealContract[string]:
		return c.Provider().ContentType()
	default:
		return "application/octet-stream"
	}
}

// CerealLegacyAdapter adapts cereal to the legacy Serializer interface
type CerealLegacyAdapter struct {
	provider cereal.CerealProvider
}

// NewCerealLegacyAdapter creates a legacy adapter for cereal
func NewCerealLegacyAdapter(provider cereal.CerealProvider) *CerealLegacyAdapter {
	return &CerealLegacyAdapter{
		provider: provider,
	}
}

// Marshal implements LegacySerializer
func (cla *CerealLegacyAdapter) Marshal(v interface{}) ([]byte, error) {
	return cla.provider.Marshal(v)
}

// Unmarshal implements LegacySerializer
func (cla *CerealLegacyAdapter) Unmarshal(data []byte, v interface{}) error {
	return cla.provider.Unmarshal(data, v)
}

// ContentType implements LegacySerializer
func (cla *CerealLegacyAdapter) ContentType() string {
	return cla.provider.ContentType()
}

// SetupCerealIntegration configures cache to use cereal for serialization
func SetupCerealIntegration(defaultFormat string) error {
	config := cereal.DefaultConfig()
	config.Name = "cache-serializer"
	config.DefaultFormat = defaultFormat
	config.EnableCaching = true
	config.EnableScoping = false // Cache doesn't need scoping
	
	// Set up the appropriate provider based on default format
	var contract interface{}
	switch defaultFormat {
	case "json":
		contract = cereal.NewJSONProvider(config)
	case "raw":
		contract = cereal.NewRawProvider(config)
	case "string":
		contract = cereal.NewStringProvider(config)
	default:
		// Default to JSON
		contract = cereal.NewJSONProvider(config)
	}
	
	// Register the contract as the global cereal singleton
	switch c := contract.(type) {
	case *cereal.CerealContract[*json.Encoder]:
		return c.Register()
	case *cereal.CerealContract[[]byte]:
		return c.Register()
	case *cereal.CerealContract[string]:
		return c.Register()
	default:
		return fmt.Errorf("unsupported contract type: %T", c)
	}
}

// Direct cereal integration functions for cache operations

// CerealMarshal provides direct access to cereal marshaling for cache
func CerealMarshal(data interface{}) ([]byte, error) {
	return cereal.Marshal(data)
}

// CerealUnmarshal provides direct access to cereal unmarshaling for cache
func CerealUnmarshal(data []byte, target interface{}) error {
	return cereal.Unmarshal(data, target)
}

// CerealMarshalWithType uses type-specific serialization
func CerealMarshalWithType[T any](data T) ([]byte, error) {
	// Use reflection to determine appropriate provider
	t := reflect.TypeOf(data)
	
	switch t.Kind() {
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			// []byte - use raw provider
			config := cereal.DefaultConfig()
			config.DefaultFormat = "raw"
			contract := cereal.NewRawProvider(config)
			return contract.Marshal(data)
		}
		fallthrough
	case reflect.String:
		// string - use string provider
		config := cereal.DefaultConfig()
		config.DefaultFormat = "string"
		contract := cereal.NewStringProvider(config)
		return contract.Marshal(data)
	default:
		// Everything else - use JSON
		return cereal.Marshal(data)
	}
}

// CerealUnmarshalWithType uses type-specific deserialization
func CerealUnmarshalWithType[T any](data []byte, target *T) error {
	// Use reflection to determine appropriate provider
	t := reflect.TypeOf(target).Elem()
	
	switch t.Kind() {
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			// []byte - use raw provider
			config := cereal.DefaultConfig()
			config.DefaultFormat = "raw"
			contract := cereal.NewRawProvider(config)
			return contract.Unmarshal(data, target)
		}
		fallthrough
	case reflect.String:
		// string - use string provider
		config := cereal.DefaultConfig()
		config.DefaultFormat = "string"
		contract := cereal.NewStringProvider(config)
		return contract.Unmarshal(data, target)
	default:
		// Everything else - use JSON
		return cereal.Unmarshal(data, target)
	}
}