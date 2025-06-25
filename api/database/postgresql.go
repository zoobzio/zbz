package database

import (
	"context"
	"fmt"
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"zbz/zlog"
)

// PostgreSQLDriver implements DatabaseDriver for PostgreSQL
type PostgreSQLDriver struct {
	*sqlx.DB
	dsn                string
	preparedStatements map[string]*sqlx.NamedStmt
}


// NewPostgreSQLDriver creates a new PostgreSQL driver
func NewPostgreSQLDriver(dsn string) (*PostgreSQLDriver, error) {
	zlog.Zlog.Debug("Initializing PostgreSQL driver", zlog.String("dsn", dsn))

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	driver := &PostgreSQLDriver{
		DB:                 db,
		dsn:                dsn,
		preparedStatements: make(map[string]*sqlx.NamedStmt),
	}

	zlog.Zlog.Info("PostgreSQL driver initialized", zlog.String("dsn", dsn))
	return driver, nil
}

// Query executes a query and returns rows
func (p *PostgreSQLDriver) Query(ctx context.Context, sql string, params map[string]any) (*sqlx.Rows, error) {
	zlog.Zlog.Debug("Executing PostgreSQL query", zlog.String("sql", sql))
	return p.NamedQueryContext(ctx, sql, params)
}


// Exec executes a statement without returning rows
func (p *PostgreSQLDriver) Exec(ctx context.Context, sql string, params map[string]any) error {
	zlog.Zlog.Debug("Executing PostgreSQL statement", zlog.String("sql", sql))
	_, err := p.NamedExecContext(ctx, sql, params)
	return err
}

// Prepare creates a prepared statement
func (p *PostgreSQLDriver) Prepare(name string, sql string) error {
	zlog.Zlog.Debug("Preparing PostgreSQL statement", 
		zlog.String("name", name), 
		zlog.String("sql", sql))
	
	stmt, err := p.PrepareNamed(sql)
	if err != nil {
		return fmt.Errorf("failed to prepare statement %s: %w", name, err)
	}
	
	p.preparedStatements[name] = stmt
	return nil
}

// ExecutePrepared executes a prepared statement
func (p *PostgreSQLDriver) ExecutePrepared(name string, params map[string]any) (*sqlx.Rows, error) {
	stmt, exists := p.preparedStatements[name]
	if !exists {
		return nil, fmt.Errorf("prepared statement %s not found", name)
	}
	
	zlog.Zlog.Debug("Executing prepared PostgreSQL statement", zlog.String("name", name))
	return stmt.Queryx(params)
}

// ReleasePrepared releases a prepared statement
func (p *PostgreSQLDriver) ReleasePrepared(name string) error {
	stmt, exists := p.preparedStatements[name]
	if !exists {
		return nil // Already released
	}
	
	err := stmt.Close()
	delete(p.preparedStatements, name)
	
	zlog.Zlog.Debug("Released prepared PostgreSQL statement", zlog.String("name", name))
	return err
}

// BeginTx starts a transaction
func (p *PostgreSQLDriver) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return p.BeginTxx(ctx, &sql.TxOptions{})
}

// Ping tests the database connection
func (p *PostgreSQLDriver) Ping(ctx context.Context) error {
	return p.PingContext(ctx)
}

// Close closes the database connection
func (p *PostgreSQLDriver) Close() error {
	// Close all prepared statements
	for name, stmt := range p.preparedStatements {
		if err := stmt.Close(); err != nil {
			zlog.Zlog.Warn("Failed to close prepared statement", 
				zlog.String("name", name), 
				zlog.Err(err))
		}
	}
	
	return p.DB.Close()
}

// DriverName returns the driver name
func (p *PostgreSQLDriver) DriverName() string {
	return "postgresql"
}

// DriverVersion returns the driver version
func (p *PostgreSQLDriver) DriverVersion() string {
	return "1.0.0" // Could query actual PostgreSQL version
}


