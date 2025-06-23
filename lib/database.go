package zbz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"zbz/lib/database"
	"zbz/shared/logger"
)

// Common database errors
var (
	ErrMacroNotFound = errors.New("macro not found")
)

// DatabaseDriver is aliased from database package
type DatabaseDriver = database.DatabaseDriver



// Database is the ZBZ service that handles business logic using a driver
type Database interface {
	// High-level operations that understand ZBZ concepts
	Execute(contract *MacroContract, params any) (*sqlx.Rows, error)
	ExecuteOnce(contract *MacroContract, params any) (*sqlx.Rows, error)
	Prepare(contract *MacroContract) error
	Dismiss(contract *MacroContract) error
	
	// Schema management
	CreateTable(meta *Meta) error
	GetSchema() Schema
	AddMeta(meta *Meta)
	
	// Macro management
	LoadMacros(macrosDir string) error
	
	// Service metadata
	ContractName() string
	ContractDescription() string
}

// zDatabase implements Database service using a DatabaseDriver
type zDatabase struct {
	driver              DatabaseDriver
	contractName        string
	contractDescription string
	schema              Schema
	macros              map[string]Macro
	preparedStatements  map[string]bool
}

// NewDatabase creates a Database service with the provided driver
func NewDatabase(driver DatabaseDriver, name, description string) Database {
	logger.Info("Creating database service", 
		logger.String("name", name),
		logger.String("driver", driver.DriverName()))
	
	return &zDatabase{
		driver:              driver,
		contractName:        name,
		contractDescription: description,
		schema:              NewSchema(),
		macros:              make(map[string]Macro),
		preparedStatements:  make(map[string]bool),
	}
}

// Execute resolves a macro contract and executes it using the driver
func (db *zDatabase) Execute(contract *MacroContract, params any) (*sqlx.Rows, error) {
	// Get the macro
	macro, exists := db.macros[contract.Macro]
	if !exists {
		logger.Error("Macro not found", logger.String("macro", contract.Macro))
		return nil, ErrMacroNotFound
	}
	
	// Interpolate the macro with embeds to get final SQL
	sql, err := macro.Interpolate(contract.Embed)
	if err != nil {
		logger.Error("Failed to interpolate macro", 
			logger.String("macro", contract.Macro),
			logger.Err(err))
		return nil, err
	}
	
	// Convert params to map[string]any
	paramMap, err := structToMap(params)
	if err != nil {
		return nil, err
	}
	
	// Execute using the driver
	ctx := context.Background()
	logger.Debug("Executing SQL", 
		logger.String("contract", contract.Name),
		logger.String("sql", sql))
	
	return db.driver.Query(ctx, sql, paramMap)
}

// ExecuteOnce executes a macro contract without preparing it
func (db *zDatabase) ExecuteOnce(contract *MacroContract, params any) (*sqlx.Rows, error) {
	// ExecuteOnce uses the same macro system as Execute
	return db.Execute(contract, params)
}

// Prepare creates a prepared statement for a macro contract
func (db *zDatabase) Prepare(contract *MacroContract) error {
	// Get the macro
	macro, exists := db.macros[contract.Macro]
	if !exists {
		return ErrMacroNotFound
	}
	
	// Interpolate the macro to get final SQL
	sql, err := macro.Interpolate(contract.Embed)
	if err != nil {
		return err
	}
	
	// Prepare using the driver
	err = db.driver.Prepare(contract.Name, sql)
	if err != nil {
		return err
	}
	
	// Track prepared statement
	db.preparedStatements[contract.Name] = true
	
	logger.Debug("Prepared statement", 
		logger.String("contract", contract.Name),
		logger.String("sql", sql))
	
	return nil
}

// Dismiss releases a prepared statement
func (db *zDatabase) Dismiss(contract *MacroContract) error {
	delete(db.preparedStatements, contract.Name)
	return db.driver.ReleasePrepared(contract.Name)
}

// CreateTable creates a table using the driver
func (db *zDatabase) CreateTable(meta *Meta) error {
	// This would use the driver to create tables
	// Implementation depends on schema building logic
	return nil
}

// GetSchema returns the database schema
func (db *zDatabase) GetSchema() Schema {
	return db.schema
}

// AddMeta adds metadata to the schema
func (db *zDatabase) AddMeta(meta *Meta) {
	db.schema.AddMeta(meta)
}

// ContractName returns the service name
func (db *zDatabase) ContractName() string {
	return db.contractName
}

// ContractDescription returns the service description
func (db *zDatabase) ContractDescription() string {
	return db.contractDescription
}

// LoadMacros loads SQL macros from the macros directory
func (db *zDatabase) LoadMacros(macrosDir string) error {
	macros, err := LoadMacrosFromDirectory(macrosDir)
	if err != nil {
		return err
	}
	
	db.macros = macros
	logger.Info("Loaded macros", logger.Int("count", len(macros)))
	return nil
}

// LoadMacrosFromDirectory loads SQL macros from a directory
func LoadMacrosFromDirectory(macrosDir string) (map[string]Macro, error) {
	macros := make(map[string]Macro)
	
	// Check if directory exists
	if _, err := os.Stat(macrosDir); os.IsNotExist(err) {
		return macros, fmt.Errorf("macros directory does not exist: %s", macrosDir)
	}
	
	// Walk through all .sqlx files in the directory
	err := filepath.Walk(macrosDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Only process .sqlx files
		if !strings.HasSuffix(info.Name(), ".sqlx") {
			return nil
		}
		
		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read macro file %s: %w", path, err)
		}
		
		// Use filename (without extension) as macro name
		macroName := strings.TrimSuffix(info.Name(), ".sqlx")
		macros[macroName] = NewMacro(macroName, string(content))
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to load macros from directory %s: %w", macrosDir, err)
	}
	
	return macros, nil
}

// Helper function to convert struct to map
func structToMap(obj any) (map[string]any, error) {
	// This would use reflection to convert struct to map
	// For now, assume it's already a map or implement reflection logic
	if m, ok := obj.(map[string]any); ok {
		return m, nil
	}
	// TODO: Add reflection-based struct to map conversion
	return make(map[string]any), nil
}