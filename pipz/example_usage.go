package pipz

import (
	"reflect"
	"strings"
)

// Example: How catalog would use pipz for metadata processing

// MetadataResult represents what an adapter extracts from a struct field
type MetadataResult struct {
	AdapterName string
	Data        any
}

// Example metadata processors that adapters would register

// CerealMetadata represents what cereal adapter would extract
type CerealMetadata struct {
	Scopes    []string
	Redaction string
	Encrypt   string
}

// ZlogMetadata represents what zlog adapter would extract  
type ZlogMetadata struct {
	Sensitive bool
	Level     string
	Exclude   bool
}

// RoccoMetadata represents what rocco adapter would extract
type RoccoMetadata struct {
	AuthRequired bool
	Roles        []string
	Permissions  []string
}

// AdapterType represents different metadata adapters
type AdapterType string

const (
	CerealAdapter AdapterType = "cereal"
	ZlogAdapter   AdapterType = "zlog"
	RoccoAdapter  AdapterType = "rocco"
)

// Example usage showing catalog with type-based contracts
func ExampleCatalogUsage() {
	// Get type-safe contract: AdapterType key, StructField input, MetadataResult output
	metadataContract := GetContract[AdapterType, reflect.StructField, MetadataResult]()
	
	// Register processors using typed keys (no magic strings!)
	metadataContract.Register(CerealAdapter, func(field reflect.StructField) MetadataResult {
		var scopes []string
		if scopeTag := field.Tag.Get("scope"); scopeTag != "" {
			scopes = strings.Split(scopeTag, ",")
		}
		
		return MetadataResult{
			AdapterName: string(CerealAdapter),
			Data: CerealMetadata{
				Scopes:    scopes,
				Redaction: field.Tag.Get("redact"),
				Encrypt:   field.Tag.Get("encrypt"),
			},
		}
	})
	
	metadataContract.Register(ZlogAdapter, func(field reflect.StructField) MetadataResult {
		logTag := field.Tag.Get("log")
		return MetadataResult{
			AdapterName: string(ZlogAdapter),
			Data: ZlogMetadata{
				Sensitive: logTag == "sensitive",
				Level:     field.Tag.Get("log_level"),
				Exclude:   logTag == "-",
			},
		}
	})
	
	metadataContract.Register(RoccoAdapter, func(field reflect.StructField) MetadataResult {
		var roles []string
		if roleTag := field.Tag.Get("role"); roleTag != "" {
			roles = strings.Split(roleTag, ",")
		}
		
		return MetadataResult{
			AdapterName: string(RoccoAdapter),
			Data: RoccoMetadata{
				AuthRequired: field.Tag.Get("auth") == "required",
				Roles:        roles,
				Permissions:  strings.Split(field.Tag.Get("permission"), ","),
			},
		}
	})
	
	// Example struct field
	type User struct {
		Email string `json:"email" scope:"private" log:"sensitive" auth:"required" role:"user,admin"`
	}
	
	// Get field for processing
	userType := reflect.TypeOf(User{})
	emailField := userType.Field(0)
	
	// 100% type-safe processing with typed keys
	cerealResult, _ := metadataContract.Process(CerealAdapter, emailField)
	zlogResult, _ := metadataContract.Process(ZlogAdapter, emailField)
	roccoResult, _ := metadataContract.Process(RoccoAdapter, emailField)
	
	_ = cerealResult // MetadataResult with CerealMetadata
	_ = zlogResult   // MetadataResult with ZlogMetadata  
	_ = roccoResult  // MetadataResult with RoccoMetadata
}

// FieldType represents zlog's field types  
type FieldType string

const (
	SecretField FieldType = "secret"
	PIIField    FieldType = "pii"
	PlainField  FieldType = "plain"
)

// Example: How zlog would use pipz with type-based contracts
func ExampleZlogUsage() {
	// Field represents zlog's current Field type
	type Field struct {
		Key   string
		Type  FieldType  // Now typed!
		Value any
	}
	
	// Get type-safe contract: FieldType key, Field input, []Field output
	fieldContract := GetContract[FieldType, Field, []Field]()
	
	// Register processors using typed keys (no magic strings!)
	fieldContract.Register(SecretField, func(field Field) []Field {
		// Redact secrets
		redacted := field
		redacted.Value = "[REDACTED]"
		return []Field{redacted}
	})
	
	fieldContract.Register(PIIField, func(field Field) []Field {
		// Hash PII
		hashed := field
		hashed.Value = "***HASHED***"
		return []Field{hashed}
	})
	
	fieldContract.Register(PlainField, func(field Field) []Field {
		// Pass through plain fields
		return []Field{field}
	})
	
	// 100% type-safe processing
	testField := Field{Key: "password", Type: SecretField, Value: "mysecret"}
	
	// Process specific field type (you know the type!)
	result, exists := fieldContract.Process(testField.Type, testField)
	if exists {
		_ = result // []Field with processed content
	}
	
	// Check what processors are available
	availableTypes := fieldContract.ListKeys() // []FieldType
	_ = availableTypes
}