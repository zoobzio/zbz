package cereal

import (
	"zbz/catalog"
	"zbz/pipz"
)

// Field represents a single field for processing - matches zlog pattern
type Field struct {
	Key         string
	Type        FieldType
	Value       any
	Permissions []string
	Metadata    catalog.FieldMetadata
}

// FieldType enum for field processing - extensible by adapters
type FieldType string

const (
	// Core field types
	StringType  FieldType = "string"
	IntType     FieldType = "int"
	FloatType   FieldType = "float"
	BoolType    FieldType = "bool"
	SliceType   FieldType = "slice"
	MapType     FieldType = "map"
	StructType  FieldType = "struct"
	
	// Security field types (can be added by adapters)
	PIIType        FieldType = "pii"
	SecretType     FieldType = "secret"
	FinancialType  FieldType = "financial"
	MedicalType    FieldType = "medical"
)

// FieldProcessor transforms fields based on permissions and context
// Matches zlog's exact pattern for consistency
type FieldProcessor func(field Field) []Field

// ValidationProcessor validates fields and returns processed/redacted versions
type ValidationProcessor func(field Field) (Field, error)

// Global processor contracts using pipz
var (
	fieldContract      *pipz.ServiceContract[FieldType, Field, []Field]
	validationContract *pipz.ServiceContract[FieldType, Field, []Field]
)

func init() {
	fieldContract = pipz.GetContract[FieldType, Field, []Field]()
	validationContract = pipz.GetContract[FieldType, Field, []Field]()
}

// RegisterFieldProcessor registers a field processor for a specific type
// Matches zlog's registration pattern exactly
func RegisterFieldProcessor(fieldType FieldType, processor FieldProcessor) {
	pipzProcessor := pipz.Processor[Field, []Field](processor)
	fieldContract.Register(fieldType, pipzProcessor)
}

// RegisterValidationProcessor registers a validation processor for a specific type
func RegisterValidationProcessor(fieldType FieldType, processor ValidationProcessor) {
	// Convert ValidationProcessor to field processor
	fieldProc := func(field Field) []Field {
		if validated, err := processor(field); err == nil {
			return []Field{validated}
		}
		return []Field{field} // Return original on error
	}
	validationContract.Register(fieldType, fieldProc)
}

// ProcessField applies the registered processor for a field type
func ProcessField(field Field) []Field {
	if result, exists := fieldContract.Process(field.Type, field); exists {
		return result
	}
	// No processor registered, return field as-is
	return []Field{field}
}

// ValidateField applies validation processor and returns processed field
func ValidateField(field Field) (Field, error) {
	if result, exists := validationContract.Process(field.Type, field); exists {
		if len(result) > 0 {
			return result[0], nil
		}
	}
	// No validation processor, return field as-is
	return field, nil
}

// Convenience functions for creating fields

func String(key, value string, permissions ...string) Field {
	return Field{
		Key:         key,
		Type:        StringType,
		Value:       value,
		Permissions: permissions,
	}
}

func Int(key string, value int, permissions ...string) Field {
	return Field{
		Key:         key,
		Type:        IntType,
		Value:       value,
		Permissions: permissions,
	}
}

func Float(key string, value float64, permissions ...string) Field {
	return Field{
		Key:         key,
		Type:        FloatType,
		Value:       value,
		Permissions: permissions,
	}
}

func Bool(key string, value bool, permissions ...string) Field {
	return Field{
		Key:         key,
		Type:        BoolType,
		Value:       value,
		Permissions: permissions,
	}
}

// Security field types (can be enhanced by adapters)

func PII(key, value string, permissions ...string) Field {
	return Field{
		Key:         key,
		Type:        PIIType,
		Value:       value,
		Permissions: permissions,
	}
}

func Secret(key, value string, permissions ...string) Field {
	return Field{
		Key:         key,
		Type:        SecretType,
		Value:       value,
		Permissions: permissions,
	}
}

func Financial(key, value string, permissions ...string) Field {
	return Field{
		Key:         key,
		Type:        FinancialType,
		Value:       value,
		Permissions: permissions,
	}
}