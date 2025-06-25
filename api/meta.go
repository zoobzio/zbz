package zbz

import (
	"maps"
	"reflect"
	"strconv"
	"strings"
	"time"

	"zbz/zlog"
)

// Meta defines the metadata for a core resource, including its name, description, and example.
// This struct serves dual purpose: table-level metadata (when Name is set) and field-level metadata.
type Meta struct {
	// Name is the table/model name (for table-level metadata) or Go field name (for field-level metadata)
	Name string

	// DatabaseColumnName is the database column name (from `db` tag)
	DatabaseColumnName string

	// JSONFieldName is the JSON field name after serialization (from `json` tag)
	JSONFieldName string

	// Description provides a human-readable description
	Description string

	// GoType is the Go type of the field, such as int, string, time.Time, etc.
	GoType string

	// DatabaseType is the SQL data type for the database column
	DatabaseType string

	// ExampleValue provides an example value for the field
	ExampleValue any

	// IsRequired indicates whether the field is required (from validation rules)
	IsRequired bool

	// ValidationRules contains validation rules string (from `validate` tag)
	ValidationRules string

	// EditType indicates how the field should be edited (from `edit` tag)
	EditType string

	// ScopeRules contains the scope requirements for this field (from `scope` tag)
	ScopeRules string

	// ColumnNames is a list of database column names (for table-level metadata)
	ColumnNames []string

	// FieldMetadata is a slice of Meta for each field in the model (for table-level metadata)
	FieldMetadata []*Meta
}

// extractFields extracts fields from a given type and returns metadata about them.
func extractFields(reflectType reflect.Type) ([]*Meta, []string, map[string]any) {
	zlog.Zlog.Debug("Extracting fields from type", 
		zlog.String("type_name", reflectType.Name()),
		zlog.Int("field_count", reflectType.NumField()))

	fieldMetas := make([]*Meta, 0, reflectType.NumField())
	columnNames := []string{}
	exampleValues := make(map[string]any)
	
	for i := range reflectType.NumField() {
		field := reflectType.Field(i)

		// Extract struct tags with descriptive names
		fieldName := field.Name
		dbColumnName := field.Tag.Get("db")
		jsonFieldName := field.Tag.Get("json")
		description := field.Tag.Get("desc")
		validationRules := field.Tag.Get("validate")
		editType := field.Tag.Get("edit")
		scopeRules := field.Tag.Get("scope")
		exampleValue := field.Tag.Get("ex")
		goFieldType := field.Type.String()
		sqlColumnType := "text" // default

		zlog.Zlog.Debug("Processing field", 
			zlog.String("field_name", fieldName),
			zlog.String("go_type", goFieldType),
			zlog.String("db_column", dbColumnName),
			zlog.String("json_name", jsonFieldName))

		// Skip fields with json:"-" as they should not appear in OpenAPI schemas
		if jsonFieldName == "-" {
			zlog.Zlog.Debug("Skipping field with json:\"-\"", zlog.String("field", fieldName))
			continue
		}

		var parsedExample any
		switch goFieldType {
		case "zbz.Model":
			// Skip the base model fields, these are handled separately
			zlog.Zlog.Debug("Skipping embedded Model field", zlog.String("field", fieldName))
			continue
		case "int", "int32":
			if exampleValue != "" {
				if parsed, err := strconv.Atoi(exampleValue); err != nil {
					zlog.Zlog.Warn("Failed to parse int example value", 
						zlog.String("field", fieldName),
						zlog.String("example", exampleValue),
						zlog.Err(err))
					parsedExample = 0
				} else {
					parsedExample = parsed
				}
			} else {
				parsedExample = 0
			}
			sqlColumnType = "integer"
		case "int64":
			if exampleValue != "" {
				if parsed, err := strconv.ParseInt(exampleValue, 10, 64); err != nil {
					zlog.Zlog.Warn("Failed to parse int64 example value", 
						zlog.String("field", fieldName),
						zlog.String("example", exampleValue),
						zlog.Err(err))
					parsedExample = int64(0)
				} else {
					parsedExample = parsed
				}
			} else {
				parsedExample = int64(0)
			}
			sqlColumnType = "bigint"
		case "float32":
			if exampleValue != "" {
				if parsed, err := strconv.ParseFloat(exampleValue, 32); err != nil {
					zlog.Zlog.Warn("Failed to parse float32 example value", 
						zlog.String("field", fieldName),
						zlog.String("example", exampleValue),
						zlog.Err(err))
					parsedExample = float32(0)
				} else {
					parsedExample = float32(parsed)
				}
			} else {
				parsedExample = float32(0)
			}
			sqlColumnType = "real"
		case "float64":
			if exampleValue != "" {
				if parsed, err := strconv.ParseFloat(exampleValue, 64); err != nil {
					zlog.Zlog.Warn("Failed to parse float64 example value", 
						zlog.String("field", fieldName),
						zlog.String("example", exampleValue),
						zlog.Err(err))
					parsedExample = float64(0)
				} else {
					parsedExample = parsed
				}
			} else {
				parsedExample = float64(0)
			}
			sqlColumnType = "double precision"
		case "string":
			parsedExample = exampleValue
			sqlColumnType = "text"
		case "bool":
			if exampleValue != "" {
				if parsed, err := strconv.ParseBool(exampleValue); err != nil {
					zlog.Zlog.Warn("Failed to parse bool example value", 
						zlog.String("field", fieldName),
						zlog.String("example", exampleValue),
						zlog.Err(err))
					parsedExample = false
				} else {
					parsedExample = parsed
				}
			} else {
				parsedExample = false
			}
			sqlColumnType = "boolean"
		case "time.Time":
			if exampleValue != "" {
				if parsed, err := time.Parse(time.RFC3339, exampleValue); err != nil {
					zlog.Zlog.Warn("Failed to parse time.Time example value", 
						zlog.String("field", fieldName),
						zlog.String("example", exampleValue),
						zlog.Err(err))
					parsedExample = time.Time{}
				} else {
					parsedExample = parsed
				}
			} else {
				parsedExample = time.Time{}
			}
			sqlColumnType = "timestamp with time zone"
		case "[]byte":
			parsedExample = []byte(exampleValue)
			sqlColumnType = "bytea"
		default:
			zlog.Zlog.Warn("Unknown Go type encountered during field extraction", 
				zlog.String("field", fieldName),
				zlog.String("go_type", goFieldType))
			parsedExample = exampleValue
		}

		fieldMeta := &Meta{
			Name:               fieldName,
			DatabaseColumnName: dbColumnName,
			JSONFieldName:      jsonFieldName,
			Description:        description,
			GoType:             goFieldType,
			DatabaseType:       sqlColumnType,
			IsRequired:         strings.Contains(validationRules, "required"),
			ValidationRules:    validationRules,
			EditType:           editType,
			ScopeRules:         scopeRules,
			ExampleValue:       parsedExample,
		}

		fieldMetas = append(fieldMetas, fieldMeta)
		
		if dbColumnName != "-" {
			columnNames = append(columnNames, dbColumnName)
			exampleValues[dbColumnName] = parsedExample
		}

		zlog.Zlog.Debug("Successfully processed field", 
			zlog.String("field", fieldName),
			zlog.String("sql_type", sqlColumnType),
			zlog.Bool("required", fieldMeta.IsRequired))
	}

	zlog.Zlog.Debug("Field extraction complete", 
		zlog.String("type", reflectType.Name()),
		zlog.Int("extracted_fields", len(fieldMetas)),
		zlog.Int("db_columns", len(columnNames)))

	return fieldMetas, columnNames, exampleValues
}

// extractMeta extracts metadata from a given model type T, which must implement BaseModel.
func extractMeta[T BaseModel](description string) *Meta {
	var modelInstance T
	modelType := reflect.TypeOf(modelInstance)

	var baseModel Model
	baseModelType := reflect.TypeOf(baseModel)

	if modelType == nil {
		zlog.Zlog.Error("Cannot extract meta from nil type")
		return nil
	}

	zlog.Zlog.Info("Starting meta extraction for model", 
		zlog.String("model_name", modelType.Name()),
		zlog.String("description", description))

	tableMeta := &Meta{
		Name:          modelType.Name(),
		Description:   description,
		FieldMetadata: make([]*Meta, 0, modelType.NumField()+baseModelType.NumField()),
		ColumnNames:   make([]string, 0, modelType.NumField()+baseModelType.NumField()),
	}

	// Extract fields from the model type
	zlog.Zlog.Debug("Extracting fields from custom model type", zlog.String("type", modelType.Name()))
	modelFields, modelColumns, modelExamples := extractFields(modelType)
	
	// Extract fields from the base Model type  
	zlog.Zlog.Debug("Extracting fields from base Model type")
	baseFields, baseColumns, baseExamples := extractFields(baseModelType)

	// Combine model and base fields
	tableMeta.FieldMetadata = append(tableMeta.FieldMetadata, modelFields...)
	tableMeta.FieldMetadata = append(tableMeta.FieldMetadata, baseFields...)

	tableMeta.ColumnNames = append(tableMeta.ColumnNames, modelColumns...)
	tableMeta.ColumnNames = append(tableMeta.ColumnNames, baseColumns...)

	// Merge example values (base examples won't override model examples)
	combinedExamples := make(map[string]any)
	maps.Copy(combinedExamples, modelExamples)
	maps.Copy(combinedExamples, baseExamples)
	tableMeta.ExampleValue = combinedExamples

	zlog.Zlog.Info("Meta extraction completed", 
		zlog.String("model_name", modelType.Name()),
		zlog.Int("total_fields", len(tableMeta.FieldMetadata)),
		zlog.Int("total_columns", len(tableMeta.ColumnNames)),
		zlog.Int("example_values", len(combinedExamples)))

	return tableMeta
}
