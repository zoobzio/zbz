package zbz

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Database provides methods to interact with the database.
type Database interface {
	Prepare(contract *MacroContract) error
	Execute(contract *MacroContract, params any) (*sqlx.Rows, error)
	Dismiss(contract *MacroContract) error
}

// zDatabase holds the configuration for the database connection.
type zDatabase struct {
	*sqlx.DB

	macros     map[string]Macro
	statements map[string]*sqlx.NamedStmt
}

// NewDatabase initializes a new Database instance with the provided configuration.
func NewDatabase() Database {
	Log.Debug("Initializing database connection")

	dsn := config.DSN()

	cx, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		Log.Fatalw("Failed to connect to database", "error", err)
	}

	db := &zDatabase{
		DB:         cx,
		macros:     make(map[string]Macro),
		statements: make(map[string]*sqlx.NamedStmt),
	}

	db.LoadTemplates("lib/macros")

	return db
}

// LoadTemplates loads SQLx macros from the specified directory into the database instance.
func (d *zDatabase) LoadTemplates(dir string) {
	Log.Infow("Loading database query macros", "query_dir", dir)

	files, err := os.ReadDir(dir)
	if err != nil {
		Log.Fatalw("Failed to read query macros directory", "error", err)
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
			Log.Fatalw("Failed to read query template file", "query_template", fullPath, "error", err)
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
