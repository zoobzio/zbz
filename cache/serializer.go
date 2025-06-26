package cache

import (
	"encoding/json"
	"fmt"
)

// Serializer handles type-safe struct ↔ []byte conversion
type Serializer[T any] interface {
	Marshal(v T) ([]byte, error)
	Unmarshal(data []byte, v *T) error
	ContentType() string
}

// SerializerManager manages type-based serializer selection (like flux adapters)
type SerializerManager struct {
	defaultFormat string
}

// NewSerializerManager creates a new serializer manager
func NewSerializerManager(defaultFormat string) *SerializerManager {
	return &SerializerManager{
		defaultFormat: defaultFormat,
	}
}

// ForType returns a type-specific serializer based on type T (like flux selectParserForKey)
func (sm *SerializerManager) ForType[T any]() Serializer[T] {
	switch any(*new(T)).(type) {
	case []byte:
		return &TypedByteSerializer[T]{}      // Pass-through for raw data
	case string:
		return &TypedStringSerializer[T]{}    // UTF-8 string handling
	default:
		// Struct types use JSON by default
		return &TypedJSONSerializer[T]{}
	}
}

// Legacy Serializer interface for backward compatibility
type LegacySerializer interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
	ContentType() string
}

// JSONSerializer implements Serializer using JSON encoding
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSON serializer
func NewJSONSerializer() LegacySerializer {
	return &JSONSerializer{}
}

func (j *JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (j *JSONSerializer) ContentType() string {
	return "application/json"
}

// ByteSerializer passes through []byte without encoding (for raw operations)
type ByteSerializer struct{}

// NewByteSerializer creates a new byte pass-through serializer
func NewByteSerializer() LegacySerializer {
	return &ByteSerializer{}
}

func (b *ByteSerializer) Marshal(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil
	default:
		return nil, fmt.Errorf("ByteSerializer only supports []byte and string, got %T", v)
	}
}

func (b *ByteSerializer) Unmarshal(data []byte, v interface{}) error {
	switch ptr := v.(type) {
	case *[]byte:
		*ptr = data
		return nil
	case *string:
		*ptr = string(data)
		return nil
	default:
		return fmt.Errorf("ByteSerializer only supports *[]byte and *string, got %T", v)
	}
}

func (b *ByteSerializer) ContentType() string {
	return "application/octet-stream"
}

// Type-safe generic serializers for V3 architecture

// JSONSerializer implements Serializer[T] using JSON encoding for struct types
type TypedJSONSerializer[T any] struct{}

func (j *TypedJSONSerializer[T]) Marshal(v T) ([]byte, error) {
	return json.Marshal(v)
}

func (j *TypedJSONSerializer[T]) Unmarshal(data []byte, v *T) error {
	return json.Unmarshal(data, v)
}

func (j *TypedJSONSerializer[T]) ContentType() string {
	return "application/json"
}

// StringSerializer implements Serializer[T] for string types
type TypedStringSerializer[T any] struct{}

func (s *TypedStringSerializer[T]) Marshal(v T) ([]byte, error) {
	// T must be string type due to ForType selection
	str, ok := any(v).(string)
	if !ok {
		return nil, fmt.Errorf("StringSerializer expects string type, got %T", v)
	}
	return []byte(str), nil
}

func (s *TypedStringSerializer[T]) Unmarshal(data []byte, v *T) error {
	// T must be string type due to ForType selection
	str := string(data)
	*v = any(str).(T)
	return nil
}

func (s *TypedStringSerializer[T]) ContentType() string {
	return "text/plain; charset=utf-8"
}

// ByteSerializer implements Serializer[T] for []byte types (pass-through)
type TypedByteSerializer[T any] struct{}

func (b *TypedByteSerializer[T]) Marshal(v T) ([]byte, error) {
	// T must be []byte type due to ForType selection
	bytes, ok := any(v).([]byte)
	if !ok {
		return nil, fmt.Errorf("ByteSerializer expects []byte type, got %T", v)
	}
	return bytes, nil
}

func (b *TypedByteSerializer[T]) Unmarshal(data []byte, v *T) error {
	// T must be []byte type due to ForType selection
	*v = any(data).(T)
	return nil
}

func (b *TypedByteSerializer[T]) ContentType() string {
	return "application/octet-stream"
}