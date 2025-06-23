package zbz

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// DatabaseConstraints represents database-level constraints derived from validation rules
type DatabaseConstraints struct {
	NotNull    bool
	Unique     bool
	Check      string
	Default    string
	Constraint string
}

// Database provides methods to interact with the database.
type Database interface {
	Prepare(contract *MacroContract) error
	Execute(contract *MacroContract, params any) (*sqlx.Rows, error)
	Dismiss(contract *MacroContract) error
	
	// One-shot execution: prepare, execute, dismiss in one call
	ExecuteOnce(contract *MacroContract, params any) (*sqlx.Rows, error)
	
	// Schema generation methods
	CreateTableFromMeta(meta *Meta) error
	DropTable(tableName string) error
	TableExists(tableName string) (bool, error)
	
	// Schema access for validation
	GetSchema() Schema
}

// zDatabase holds the configuration for the database connection.
type zDatabase struct {
	*sqlx.DB

	macros     map[string]Macro
	statements map[string]*sqlx.NamedStmt
	schema     Schema
	validator  Validate
}

// NewDatabase initializes a new Database instance.
func NewDatabase() Database {
	Log.Debug("Initializing database connection")

	dsn := config.DSN()

	cx, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		Log.Fatal("Failed to connect to database", zap.Error(err))
	}

	db := &zDatabase{
		DB:         cx,
		macros:     make(map[string]Macro),
		statements: make(map[string]*sqlx.NamedStmt),
		schema:     NewSchema(),
		validator:  NewValidate(),
	}

	db.LoadTemplates("lib/macros")

	return db
}

// LoadTemplates loads SQLx macros from the specified directory into the database instance.
func (d *zDatabase) LoadTemplates(dir string) {
	Log.Info("Loading database query macros", zap.String("query_dir", dir))

	files, err := os.ReadDir(dir)
	if err != nil {
		Log.Fatal("Failed to read query macros directory", zap.Error(err))
	}

	macros := make(map[string]Macro)
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
			Log.Fatal("Failed to read query template file", zap.String("query_template", fullPath), zap.Error(err))
		}

		macros[key] = NewMacro(key, string(content))
	}

	maps.Copy(d.macros, macros)
}

// Prepare a SQL query by its name and optional embedded parameters.
func (d *zDatabase) Prepare(contract *MacroContract) error {
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

// Execute a prepared SQL statement with the provided parameters.
func (d *zDatabase) Execute(contract *MacroContract, params any) (*sqlx.Rows, error) {
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

// Dismiss a prepared SQL statement by its name.
func (d *zDatabase) Dismiss(contract *MacroContract) error {
	stmt, ok := d.statements[contract.Name]
	if !ok {
		return fmt.Errorf("statement %s not found", contract.Name)
	}

	if err := stmt.Close(); err != nil {
		return err
	}

	return nil
}

// ExecuteOnce prepares, executes, and dismisses a statement in one shot - perfect for table creation
func (d *zDatabase) ExecuteOnce(contract *MacroContract, params any) (*sqlx.Rows, error) {
	// Prepare the statement
	err := d.Prepare(contract)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement for one-shot execution: %w", err)
	}
	
	// Always clean up, even if execution fails
	defer func() {
		if dismissErr := d.Dismiss(contract); dismissErr != nil {
			Log.Warn("Failed to dismiss statement after one-shot execution", 
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

// CreateTableFromMeta creates a database table from model metadata with validation constraints
func (d *zDatabase) CreateTableFromMeta(meta *Meta) error {
	Log.Info("Creating table from metadata", 
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
	
	Log.Debug("Executing CREATE TABLE statement", zap.String("sql", sql))
	
	// Execute the CREATE TABLE statement
	_, err := d.Exec(sql)
	if err != nil {
		Log.Error("Failed to create table", 
			zap.String("table_name", tableName),
			zap.Error(err))
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}
	
	Log.Info("Successfully created table", zap.String("table_name", tableName))
	return nil
}

// buildColumnDefinition converts a field to a PostgreSQL column definition
func (d *zDatabase) buildColumnDefinition(field *Meta) string {
	if field.JSONFieldName == "" {
		return ""
	}
	
	// Get PostgreSQL data type
	pgType := d.getPostgreSQLType(field.GoType)
	if pgType == "" {
		Log.Warn("Unknown field type for database", 
			zap.String("field", field.JSONFieldName),
			zap.String("type", field.GoType))
		return ""
	}
	
	// Build basic column definition
	column := fmt.Sprintf("%s %s", field.DatabaseColumnName, pgType)
	
	// Parse validation rules and apply database constraints
	// Parse validation rules using validator, then generate DB constraints
	rules := d.validator.ParseValidationRules(field.ValidationRules)
	parsedRules := ParsedValidationRules{
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
func (d *zDatabase) buildFieldConstraints(field *Meta) []string {
	var constraints []string
	
	rules := d.validator.ParseValidationRules(field.ValidationRules)
	parsedRules := ParsedValidationRules{
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
func (d *zDatabase) getPostgreSQLType(goType string) string {
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
func (d *zDatabase) DropTable(tableName string) error {
	Log.Info("Dropping table", zap.String("table_name", tableName))
	
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", tableName)
	
	_, err := d.Exec(sql)
	if err != nil {
		Log.Error("Failed to drop table", 
			zap.String("table_name", tableName),
			zap.Error(err))
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}
	
	Log.Info("Successfully dropped table", zap.String("table_name", tableName))
	return nil
}

// TableExists checks if a table exists in the database
func (d *zDatabase) TableExists(tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		);`
	
	err := d.Get(&exists, query, tableName)
	if err != nil {
		Log.Error("Failed to check table existence", 
			zap.String("table_name", tableName),
			zap.Error(err))
		return false, fmt.Errorf("failed to check table existence %s: %w", tableName, err)
	}
	
	return exists, nil
}

// GetSchema returns the schema instance for validation
func (d *zDatabase) GetSchema() Schema {
	return d.schema
}

// getDatabaseConstraints converts parsed validation rules to database constraints
func (d *zDatabase) getDatabaseConstraints(parsed ParsedValidationRules) DatabaseConstraints {
	constraints := DatabaseConstraints{}

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

// CreateDatabaseFromContract creates a database instance from a DatabaseContract
func CreateDatabaseFromContract(contract DatabaseContract) Database {
	Log.Debug("Creating database from contract", 
		zap.String("driver", contract.Driver),
		zap.String("name", contract.Name))
		// DSN not logged for security (contains credentials)
	
	switch contract.Driver {
	case "postgres", "postgresql":
		// Create PostgreSQL connection directly here to avoid circular imports
		Log.Debug("Initializing PostgreSQL database connection")

		cx, err := sqlx.Connect("postgres", contract.DSN)
		if err != nil {
			Log.Fatal("Failed to connect to PostgreSQL database", zap.Error(err))
		}

		// TODO: Configure connection pool from contract.Pool when needed
		// For now just use defaults

		db := &zDatabase{
			DB:         cx,
			macros:     make(map[string]Macro),
			statements: make(map[string]*sqlx.NamedStmt),
			schema:     NewSchema(),
			validator:  NewValidate(),
		}

		db.LoadTemplates("lib/macros")
		return db
	case "mysql":
		Log.Fatal("MySQL adapter not implemented yet", zap.String("driver", contract.Driver))
	case "sqlite":
		Log.Fatal("SQLite adapter not implemented yet", zap.String("driver", contract.Driver))
	default:
		Log.Fatal("Unsupported database driver", zap.String("driver", contract.Driver))
	}
	
	return nil
}

