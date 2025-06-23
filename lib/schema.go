package zbz

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zbz/shared/logger"
	"gopkg.in/yaml.v3"
)

// DatabaseSchemaDocument represents the complete database schema for AI consumption
type DatabaseSchemaDocument struct {
	Version     string                  `json:"version" yaml:"version"`
	Database    string                  `json:"database" yaml:"database"`
	Description string                  `json:"description" yaml:"description"`
	Tables      map[string]*TableSchema `json:"tables" yaml:"tables"`
	Relations   []*TableRelation        `json:"relations" yaml:"relations"`
	Indexes     []*IndexSchema          `json:"indexes,omitempty" yaml:"indexes,omitempty"`
	Constraints []*ConstraintSchema     `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Views       map[string]*ViewSchema  `json:"views,omitempty" yaml:"views,omitempty"`
	Metadata    *SchemaMetadata         `json:"metadata" yaml:"metadata"`
}

// TableSchema represents a single table's structure
type TableSchema struct {
	Name        string                   `json:"name" yaml:"name"`
	Description string                   `json:"description" yaml:"description"`
	Columns     map[string]*ColumnSchema `json:"columns" yaml:"columns"`
	PrimaryKey  []string                 `json:"primaryKey" yaml:"primaryKey"`
	Unique      [][]string               `json:"unique,omitempty" yaml:"unique,omitempty"`
	Indexes     []string                 `json:"indexes,omitempty" yaml:"indexes,omitempty"`
	Tags        []string                 `json:"tags,omitempty" yaml:"tags,omitempty"`
	Examples    map[string]any           `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// ColumnSchema represents a single column's structure
type ColumnSchema struct {
	Name         string               `json:"name" yaml:"name"`
	Type         string               `json:"type" yaml:"type"`
	SQLType      string               `json:"sqlType" yaml:"sqlType"`
	Description  string               `json:"description" yaml:"description"`
	Required     bool                 `json:"required" yaml:"required"`
	Unique       bool                 `json:"unique" yaml:"unique"`
	DefaultValue any                  `json:"defaultValue,omitempty" yaml:"defaultValue,omitempty"`
	Constraints  *ColumnConstraints   `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Validation   map[string]any       `json:"validation,omitempty" yaml:"validation,omitempty"`
	Examples     []any                `json:"examples,omitempty" yaml:"examples,omitempty"`
	Format       string               `json:"format,omitempty" yaml:"format,omitempty"`
	Enum         []any                `json:"enum,omitempty" yaml:"enum,omitempty"`
	References   *ForeignKeyReference `json:"references,omitempty" yaml:"references,omitempty"`
}

// ColumnConstraints represents validation and database constraints on a column
type ColumnConstraints struct {
	MinLength   *int     `json:"minLength,omitempty"`
	MaxLength   *int     `json:"maxLength,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	CheckClause string   `json:"checkClause,omitempty"`
}

// ForeignKeyReference represents a foreign key relationship
type ForeignKeyReference struct {
	Table    string `json:"table"`
	Column   string `json:"column"`
	OnDelete string `json:"onDelete,omitempty"`
	OnUpdate string `json:"onUpdate,omitempty"`
}

// TableRelation represents relationships between tables
type TableRelation struct {
	Type        string   `json:"type"` // "one-to-one", "one-to-many", "many-to-many"
	FromTable   string   `json:"fromTable"`
	FromColumns []string `json:"fromColumns"`
	ToTable     string   `json:"toTable"`
	ToColumns   []string `json:"toColumns"`
	Description string   `json:"description"`
	JoinTable   string   `json:"joinTable,omitempty"` // For many-to-many
}

// IndexSchema represents database indexes
type IndexSchema struct {
	Name        string   `json:"name"`
	Table       string   `json:"table"`
	Columns     []string `json:"columns"`
	Type        string   `json:"type"` // "btree", "hash", "gin", etc.
	Unique      bool     `json:"unique"`
	Description string   `json:"description,omitempty"`
}

// ConstraintSchema represents database constraints
type ConstraintSchema struct {
	Name        string `json:"name"`
	Table       string `json:"table"`
	Type        string `json:"type"` // "check", "foreign_key", "unique", etc.
	Definition  string `json:"definition"`
	Description string `json:"description,omitempty"`
}

// ViewSchema represents database views
type ViewSchema struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Columns      map[string]string `json:"columns"` // column_name -> type
	Query        string            `json:"query,omitempty"`
	Materialized bool              `json:"materialized"`
}

// SchemaMetadata contains additional information about the schema
type SchemaMetadata struct {
	GeneratedAt    string            `json:"generatedAt"`
	Generator      string            `json:"generator"`
	TotalTables    int               `json:"totalTables"`
	TotalColumns   int               `json:"totalColumns"`
	TotalRelations int               `json:"totalRelations"`
	DatabaseInfo   map[string]string `json:"databaseInfo,omitempty"`
	NLQHints       *NLQHints         `json:"nlqHints,omitempty"`
}

// NLQHints provides guidance for Natural Language Query processing
type NLQHints struct {
	CommonQueries    []string            `json:"commonQueries,omitempty"`
	SearchableFields []string            `json:"searchableFields,omitempty"`
	DateFields       []string            `json:"dateFields,omitempty"`
	NumericFields    []string            `json:"numericFields,omitempty"`
	CategoryFields   map[string][]any    `json:"categoryFields,omitempty"`
	JoinPatterns     []string            `json:"joinPatterns,omitempty"`
	Synonyms         map[string][]string `json:"synonyms,omitempty"`
}

// Schema interface for generating database schema documents
type Schema interface {
	AddMeta(meta *Meta)
	GenerateDocument() *DatabaseSchemaDocument
	
	// Framework-agnostic schema handler
	SchemaHandler(ctx RequestContext)
	
	// Handler contract for engine to collect
	SchemaContract(databaseKey string) *HandlerContract
	
	// Validation methods for SQL macro safety
	IsValidTable(name string) bool
	IsValidColumns(table string, columns []string) bool
	GetTableColumns(table string) []string
}

// zSchema implements the Schema interface
type zSchema struct {
	metas  []Meta
	schema *DatabaseSchemaDocument
}

// NewSchema creates a new Schema instance
func NewSchema() Schema {
	return &zSchema{
		metas: make([]Meta, 0),
	}
}

// AddMeta registers a model's metadata for schema generation
func (s *zSchema) AddMeta(meta *Meta) {
	logger.Log.Debug("Adding meta to schema",
		logger.Any("meta", meta))
	s.metas = append(s.metas, *meta)
	// Invalidate cached schema so it gets regenerated with new meta
	s.schema = nil
}

// GenerateDocument creates a complete database schema document
func (s *zSchema) GenerateDocument() *DatabaseSchemaDocument {
	tables := make(map[string]*TableSchema)
	relations := make([]*TableRelation, 0)
	totalColumns := 0

	// Process each registered model
	for _, meta := range s.metas {
		tableSchema := s.generateTableSchema(&meta)
		tables[strings.ToLower(meta.Name)] = tableSchema
		totalColumns += len(tableSchema.Columns)

		// Extract relationships from fields
		for _, field := range meta.FieldMetadata {
			if field.GoType == "zbz.Model" {
				continue // Skip embedded model fields
			}

			// Look for foreign key relationships based on naming patterns
			if strings.HasSuffix(field.DatabaseColumnName, "_id") && field.GoType == "string" {
				referencedTable := strings.TrimSuffix(field.DatabaseColumnName, "_id")
				relations = append(relations, &TableRelation{
					Type:        "many-to-one",
					FromTable:   strings.ToLower(meta.Name),
					FromColumns: []string{field.DatabaseColumnName},
					ToTable:     referencedTable,
					ToColumns:   []string{"id"},
					Description: fmt.Sprintf("%s belongs to %s", meta.Name, referencedTable),
				})
			}
		}
	}

	// Generate NLQ hints
	nlqHints := s.generateNLQHints(tables)

	s.schema = &DatabaseSchemaDocument{
		Version:     "1.0",
		Database:    "zbz_api",
		Description: "ZBZ Framework API Database Schema - Auto-generated schema document for AI agent consumption and Natural Language Query processing",
		Tables:      tables,
		Relations:   relations,
		Metadata: &SchemaMetadata{
			GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
			Generator:      "ZBZ Framework v1.0",
			TotalTables:    len(tables),
			TotalColumns:   totalColumns,
			TotalRelations: len(relations),
			DatabaseInfo: map[string]string{
				"dialect": "postgresql",
				"version": "15+",
			},
			NLQHints: nlqHints,
		},
	}

	return s.schema
}

// generateTableSchema creates a TableSchema from Meta
func (s *zSchema) generateTableSchema(meta *Meta) *TableSchema {
	columns := make(map[string]*ColumnSchema)
	primaryKey := []string{"id"}
	examples := make(map[string]any)

	for _, field := range meta.FieldMetadata {
		if field.GoType == "zbz.Model" || field.DatabaseColumnName == "-" {
			continue
		}

		columnSchema := &ColumnSchema{
			Name:        field.DatabaseColumnName,
			Type:        field.GoType,
			SQLType:     field.DatabaseType,
			Description: field.Description,
			Required:    field.IsRequired,
		}

		// Add validation constraints if available
		if field.ValidationRules != "" {
			constraints := s.parseValidationConstraints(field.ValidationRules)
			if constraints != nil {
				columnSchema.Constraints = constraints
			}
		}

		// Add examples
		if field.ExampleValue != nil {
			columnSchema.Examples = []any{field.ExampleValue}
			examples[field.DatabaseColumnName] = field.ExampleValue
		}

		// Detect unique fields
		if strings.Contains(field.ValidationRules, "email") || strings.Contains(field.ValidationRules, "uuid") {
			columnSchema.Unique = true
		}

		columns[field.DatabaseColumnName] = columnSchema
	}

	return &TableSchema{
		Name:        strings.ToLower(meta.Name),
		Description: meta.Description,
		Columns:     columns,
		PrimaryKey:  primaryKey,
		Examples:    examples,
		Tags:        []string{meta.Name, "api-resource"},
	}
}

// parseValidationConstraints converts validation tags to constraints
func (s *zSchema) parseValidationConstraints(validation string) *ColumnConstraints {
	if validation == "" {
		return nil
	}

	constraints := &ColumnConstraints{}
	rules := strings.Split(validation, ",")

	for _, rule := range rules {
		parts := strings.Split(strings.TrimSpace(rule), "=")
		ruleName := parts[0]

		switch ruleName {
		case "min":
			if len(parts) > 1 {
				if val := parseFloat(parts[1]); val != nil {
					constraints.Minimum = val
				}
			}
		case "max":
			if len(parts) > 1 {
				if val := parseFloat(parts[1]); val != nil {
					constraints.Maximum = val
				}
			}
		case "minlength":
			if len(parts) > 1 {
				if val := parseInt(parts[1]); val != nil {
					constraints.MinLength = val
				}
			}
		case "maxlength":
			if len(parts) > 1 {
				if val := parseInt(parts[1]); val != nil {
					constraints.MaxLength = val
				}
			}
		}
	}

	return constraints
}

// generateNLQHints creates hints for Natural Language Query processing
func (s *zSchema) generateNLQHints(tables map[string]*TableSchema) *NLQHints {
	searchableFields := make([]string, 0)
	dateFields := make([]string, 0)
	numericFields := make([]string, 0)
	categoryFields := make(map[string][]any)
	synonyms := make(map[string][]string)

	for tableName, table := range tables {
		for columnName, column := range table.Columns {
			fullField := fmt.Sprintf("%s.%s", tableName, columnName)

			// Categorize fields by type
			switch column.Type {
			case "string":
				if strings.Contains(columnName, "name") || strings.Contains(columnName, "title") ||
					strings.Contains(columnName, "description") || strings.Contains(columnName, "address") {
					searchableFields = append(searchableFields, fullField)
				}
			case "time.Time":
				dateFields = append(dateFields, fullField)
			case "int", "int64", "float64":
				numericFields = append(numericFields, fullField)
			}

			// Add synonyms for common field names
			if strings.Contains(columnName, "name") {
				synonyms[fullField] = []string{"title", "label", "identifier"}
			}
			if strings.Contains(columnName, "email") {
				synonyms[fullField] = []string{"email address", "contact email", "mail"}
			}
		}
	}

	return &NLQHints{
		CommonQueries: []string{
			"Find all users created this month",
			"Show contacts with gmail addresses",
			"List companies by creation date",
			"Get forms with more than 5 fields",
		},
		SearchableFields: searchableFields,
		DateFields:       dateFields,
		NumericFields:    numericFields,
		CategoryFields:   categoryFields,
		JoinPatterns: []string{
			"users -> contacts (user_id)",
			"companies -> contacts (company_id)",
			"forms -> fields (form_id)",
			"fields -> properties (field_id)",
		},
		Synonyms: synonyms,
	}
}

// SchemaHandler serves the database schema document as YAML for browser display
func (s *zSchema) SchemaHandler(ctx RequestContext) {
	if s.schema == nil {
		s.GenerateDocument()
	}

	yamlData, err := yaml.Marshal(s.schema)
	if err != nil {
		ctx.Set("error_message", "Failed to generate YAML schema")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	// Use text/plain to display in browser instead of downloading
	ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
	ctx.SetHeader("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	ctx.Status(http.StatusOK)
	ctx.Data("text/plain; charset=utf-8", yamlData)
}

// SchemaContract returns the handler contract for schema endpoint
func (s *zSchema) SchemaContract(databaseKey string) *HandlerContract {
	path := "/schema"
	name := "Get Default Schema"
	description := "Get default database schema"
	
	if databaseKey != "primary" && databaseKey != "" {
		path = fmt.Sprintf("/schema/%s", databaseKey)
		name = fmt.Sprintf("Get %s Schema", databaseKey)
		description = fmt.Sprintf("Get database schema for %s", databaseKey)
	}
	
	return &HandlerContract{
		Name:        name,
		Description: description,
		Method:      "GET",
		Path:        path,
		Handler:     s.SchemaHandler,
		Auth:        false, // Schema is public for introspection
	}
}

// Helper functions for parsing validation rules
func parseFloat(s string) *float64 {
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return &val
	}
	return nil
}

func parseInt(s string) *int {
	if val, err := strconv.Atoi(s); err == nil {
		return &val
	}
	return nil
}

// IsValidTable checks if a table name exists in the schema
func (s *zSchema) IsValidTable(name string) bool {
	// Always generate fresh schema to avoid caching issues
	s.GenerateDocument()
	
	logger.Log.Debug("Checking table validity", 
		logger.String("table", name),
		logger.Int("total_metas", len(s.metas)),
		logger.Int("total_tables", len(s.schema.Tables)),
		logger.Strings("available_tables", s.getTableNames()),
		logger.Any("schema_metas", s.metas))
	
	_, exists := s.schema.Tables[strings.ToLower(name)]
	return exists
}

// IsValidColumns checks if all columns exist in the specified table
func (s *zSchema) IsValidColumns(table string, columns []string) bool {
	// Always generate fresh schema
	s.GenerateDocument()
	
	tableSchema, exists := s.schema.Tables[strings.ToLower(table)]
	if !exists {
		return false
	}
	
	for _, column := range columns {
		column = strings.TrimSpace(column)
		if _, exists := tableSchema.Columns[column]; !exists {
			return false
		}
	}
	
	return true
}

// GetTableColumns returns all column names for a table
func (s *zSchema) GetTableColumns(table string) []string {
	// Always generate fresh schema
	s.GenerateDocument()
	
	tableSchema, exists := s.schema.Tables[strings.ToLower(table)]
	if !exists {
		return []string{}
	}
	
	columns := make([]string, 0, len(tableSchema.Columns))
	for columnName := range tableSchema.Columns {
		columns = append(columns, columnName)
	}
	
	return columns
}

// getTableNames returns a list of all table names for debugging
func (s *zSchema) getTableNames() []string {
	names := make([]string, 0, len(s.schema.Tables))
	for name := range s.schema.Tables {
		names = append(names, name)
	}
	return names
}
