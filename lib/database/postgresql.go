package database

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	zbz "zbz/lib"
)

// PostgreSQLDatabase implements zbz.Database interface for PostgreSQL
type PostgreSQLDatabase struct {
	*sqlx.DB
	dsn        string
	macros     map[string]zbz.Macro
	statements map[string]*sqlx.NamedStmt
	schema     zbz.Schema
	validator  zbz.Validate
}

// NewPostgreSQLDatabase creates a new PostgreSQL database instance
func NewPostgreSQLDatabase(dsn string, pool any) zbz.Database {
	zbz.Log.Debug("Initializing PostgreSQL database connection", zap.String("dsn", dsn))

	cx, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		zbz.Log.Fatal("Failed to connect to PostgreSQL database", zap.Error(err))
	}

	// TODO: Configure connection pool when needed
	// For now just use defaults

	db := &PostgreSQLDatabase{
		DB:         cx,
		dsn:        dsn,
		macros:     make(map[string]zbz.Macro),
		statements: make(map[string]*sqlx.NamedStmt),
		schema:     zbz.NewSchema(),
		validator:  zbz.NewValidate(),
	}

	db.LoadTemplates("lib/macros")
	return db
}

// Driver returns the database driver name
func (d *PostgreSQLDatabase) Driver() string {
	return "postgres"
}

// ConnectionString returns the connection string
func (d *PostgreSQLDatabase) ConnectionString() string {
	return d.dsn
}

// LoadTemplates loads SQLx macros from the specified directory
func (d *PostgreSQLDatabase) LoadTemplates(dir string) {
	zbz.Log.Info("Loading PostgreSQL database query macros", zap.String("query_dir", dir))

	files, err := os.ReadDir(dir)
	if err != nil {
		zbz.Log.Fatal("Failed to read query macros directory", zap.Error(err))
	}

	macros := make(map[string]zbz.Macro)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()
		ext := filepath.Ext(filename)
		key := strings.TrimSuffix(filename, ext)
		fullPath := filepath.Join(dir, filename)

		if ext != ".sqlx" {
			continue
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			zbz.Log.Fatal("Failed to read query template file", zap.String("query_template", fullPath), zap.Error(err))
		}

		macros[key] = zbz.NewMacro(key, string(content))
	}

	maps.Copy(d.macros, macros)
}

// Prepare a SQL query by its name and optional embedded parameters
func (d *PostgreSQLDatabase) Prepare(contract *zbz.MacroContract) error {
	query, ok := d.macros[contract.Macro]
	if !ok {
		return fmt.Errorf("query %s not found", contract.Macro)
	}

	q, err := query.Interpolate(contract.Embed)
	if err != nil {
		return err
	}

	stmt, err := d.PrepareNamed(q)
	if err != nil {
		return err
	}

	d.statements[contract.Name] = stmt
	return nil
}

// Execute a prepared SQL statement with the provided parameters
func (d *PostgreSQLDatabase) Execute(contract *zbz.MacroContract, params any) (*sqlx.Rows, error) {
	stmt, ok := d.statements[contract.Name]
	if !ok {
		return nil, fmt.Errorf("statement %s not found", contract.Name)
	}

	rows, err := stmt.Queryx(params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement %s: %w", contract.Name, err)
	}

	return rows, nil
}

// Dismiss a prepared SQL statement by its name
func (d *PostgreSQLDatabase) Dismiss(contract *zbz.MacroContract) error {
	stmt, ok := d.statements[contract.Name]
	if !ok {
		return fmt.Errorf("statement %s not found", contract.Name)
	}

	if err := stmt.Close(); err != nil {
		return err
	}

	return nil
}

// ExecuteOnce prepares, executes, and dismisses a statement in one shot
func (d *PostgreSQLDatabase) ExecuteOnce(contract *zbz.MacroContract, params any) (*sqlx.Rows, error) {
	// Prepare the statement
	err := d.Prepare(contract)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement for one-shot execution: %w", err)
	}
	
	// Always clean up, even if execution fails
	defer func() {
		if dismissErr := d.Dismiss(contract); dismissErr != nil {
			zbz.Log.Warn("Failed to dismiss statement after one-shot execution", 
				zap.String("contract", contract.Name), 
				zap.Error(dismissErr))
		}
	}()
	
	// Handle nil params - convert to empty map for sqlx compatibility
	if params == nil {
		params = map[string]any{}
	}
	
	// Execute the statement
	rows, err := d.Execute(contract, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement in one-shot: %w", err)
	}
	
	return rows, nil
}

// GetSchema returns the schema instance for validation
func (d *PostgreSQLDatabase) GetSchema() zbz.Schema {
	return d.schema
}

// CreateTableFromMeta creates a database table from model metadata with validation constraints
func (d *PostgreSQLDatabase) CreateTableFromMeta(meta *zbz.Meta) error {
	zbz.Log.Info("Creating PostgreSQL table from metadata", 
		zap.String("table_name", meta.Name),
		zap.Int("field_count", len(meta.FieldMetadata)))
	
	// Start building CREATE TABLE statement
	var columns []string
	var constraints []string
	
	// Add standard Model fields first (id, created_at, updated_at)
	columns = append(columns, "id UUID PRIMARY KEY DEFAULT gen_random_uuid()")
	columns = append(columns, "created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()")
	columns = append(columns, "updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()")
	
	// Process each field from the metadata
	for _, field := range meta.FieldMetadata {
		column := d.buildColumnDefinition(field)
		if column != "" {
			columns = append(columns, column)
		}
		
		// Add validation constraints
		fieldConstraints := d.buildFieldConstraints(field)
		constraints = append(constraints, fieldConstraints...)
	}
	
	// Build the complete CREATE TABLE statement
	tableName := strings.ToLower(meta.Name) + "s" // Pluralize table name
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n    %s", 
		tableName, 
		strings.Join(columns, ",\n    "))
	
	// Add constraints if any
	if len(constraints) > 0 {
		sql += ",\n    " + strings.Join(constraints, ",\n    ")
	}
	
	sql += "\n);"
	
	zbz.Log.Debug("Executing PostgreSQL CREATE TABLE statement", zap.String("sql", sql))
	
	// Execute the CREATE TABLE statement
	_, err := d.Exec(sql)
	if err != nil {
		zbz.Log.Error("Failed to create PostgreSQL table", 
			zap.String("table_name", tableName),
			zap.Error(err))
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}
	
	zbz.Log.Info("Successfully created PostgreSQL table", zap.String("table_name", tableName))
	return nil
}

// buildColumnDefinition converts a field to a PostgreSQL column definition
func (d *PostgreSQLDatabase) buildColumnDefinition(field *zbz.Meta) string {
	if field.JSONFieldName == "" {
		return ""
	}
	
	// Get PostgreSQL data type
	pgType := d.getPostgreSQLType(field.GoType)
	if pgType == "" {
		zbz.Log.Warn("Unknown field type for PostgreSQL", 
			zap.String("field", field.JSONFieldName),
			zap.String("type", field.GoType))
		return ""
	}
	
	// Build basic column definition
	column := fmt.Sprintf("%s %s", field.DatabaseColumnName, pgType)
	
	// Parse validation rules and apply database constraints
	rules := d.validator.ParseValidationRules(field.ValidationRules)
	parsedRules := zbz.ParsedValidationRules{
		Rules:     rules,
		FieldType: field.GoType,
		FieldName: field.JSONFieldName,
	}
	constraints := d.getDatabaseConstraints(parsedRules)
	
	// Apply NOT NULL constraint
	if constraints.NotNull || field.IsRequired {
		column += " NOT NULL"
	}
	
	// Apply UNIQUE constraint
	if constraints.Unique {
		column += " UNIQUE"
	}
	
	// Apply DEFAULT constraint if specified
	if constraints.Default != "" {
		column += " DEFAULT " + constraints.Default
	}
	
	return column
}

// buildFieldConstraints builds CHECK constraints for field validation
func (d *PostgreSQLDatabase) buildFieldConstraints(field *zbz.Meta) []string {
	var constraints []string
	
	rules := d.validator.ParseValidationRules(field.ValidationRules)
	parsedRules := zbz.ParsedValidationRules{
		Rules:     rules,
		FieldType: field.GoType,
		FieldName: field.JSONFieldName,
	}
	dbConstraints := d.getDatabaseConstraints(parsedRules)
	
	// Add CHECK constraint if specified
	if dbConstraints.Check != "" {
		// Replace placeholder with actual column name
		checkConstraint := strings.ReplaceAll(dbConstraints.Check, "{{column}}", field.DatabaseColumnName)
		constraintName := fmt.Sprintf("check_%s_%s", field.DatabaseColumnName, "validation")
		constraints = append(constraints, fmt.Sprintf("CONSTRAINT %s CHECK (%s)", constraintName, checkConstraint))
	}
	
	return constraints
}

// getPostgreSQLType converts Go types to PostgreSQL types
func (d *PostgreSQLDatabase) getPostgreSQLType(goType string) string {
	switch goType {
	case "string":
		return "TEXT"
	case "int", "int32":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "float32":
		return "REAL"
	case "float64":
		return "DOUBLE PRECISION"
	case "bool":
		return "BOOLEAN"
	case "time.Time":
		return "TIMESTAMP WITH TIME ZONE"
	case "[]byte":
		return "BYTEA"
	default:
		// Handle pointer types
		if strings.HasPrefix(goType, "*") {
			return d.getPostgreSQLType(goType[1:])
		}
		return ""
	}
}

// DropTable drops a table from the database
func (d *PostgreSQLDatabase) DropTable(tableName string) error {
	zbz.Log.Info("Dropping PostgreSQL table", zap.String("table_name", tableName))
	
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", tableName)
	
	_, err := d.Exec(sql)
	if err != nil {
		zbz.Log.Error("Failed to drop PostgreSQL table", 
			zap.String("table_name", tableName),
			zap.Error(err))
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}
	
	zbz.Log.Info("Successfully dropped PostgreSQL table", zap.String("table_name", tableName))
	return nil
}

// TableExists checks if a table exists in the database
func (d *PostgreSQLDatabase) TableExists(tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		);`
	
	err := d.Get(&exists, query, tableName)
	if err != nil {
		zbz.Log.Error("Failed to check PostgreSQL table existence", 
			zap.String("table_name", tableName),
			zap.Error(err))
		return false, fmt.Errorf("failed to check table existence %s: %w", tableName, err)
	}
	
	return exists, nil
}

// getDatabaseConstraints converts parsed validation rules to database constraints
func (d *PostgreSQLDatabase) getDatabaseConstraints(parsed zbz.ParsedValidationRules) zbz.DatabaseConstraints {
	constraints := zbz.DatabaseConstraints{}

	for _, rule := range parsed.Rules {
		switch rule.Name {
		case "required":
			constraints.NotNull = true

		case "uuid", "uuid4", "email":
			// UUIDs and emails should typically be unique
			constraints.Unique = true

		case "min":
			if len(rule.Params) > 0 {
				if parsed.FieldType == "string" {
					constraints.Check = fmt.Sprintf("LENGTH(%s) >= %s", "{{column}}", rule.Params[0])
				} else if parsed.FieldType == "integer" || parsed.FieldType == "number" {
					constraints.Check = fmt.Sprintf("%s >= %s", "{{column}}", rule.Params[0])
				}
			}

		case "max":
			if len(rule.Params) > 0 {
				if parsed.FieldType == "string" {
					constraints.Check = fmt.Sprintf("LENGTH(%s) <= %s", "{{column}}", rule.Params[0])
				} else if parsed.FieldType == "integer" || parsed.FieldType == "number" {
					constraints.Check = fmt.Sprintf("%s <= %s", "{{column}}", rule.Params[0])
				}
			}

		case "oneof":
			if len(rule.Params) > 0 {
				values := make([]string, len(rule.Params))
				for i, param := range rule.Params {
					values[i] = fmt.Sprintf("'%s'", param)
				}
				constraints.Check = fmt.Sprintf("%s IN (%s)", "{{column}}", strings.Join(values, ", "))
			}

		case "regexp":
			if len(rule.Params) > 0 {
				constraints.Check = fmt.Sprintf("%s ~ '%s'", "{{column}}", rule.Params[0])
			}
		}
	}

	return constraints
}