package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// DatabaseDriver defines the interface that database adapters must implement
// This is what user-initialized drivers (PostgreSQL, MySQL, etc.) implement
type DatabaseDriver interface {
	// Basic query execution
	Query(ctx context.Context, sql string, params map[string]any) (*sqlx.Rows, error)
	Exec(ctx context.Context, sql string, params map[string]any) error
	
	// Prepared statement management
	Prepare(name string, sql string) error
	ExecutePrepared(name string, params map[string]any) (*sqlx.Rows, error)
	ReleasePrepared(name string) error
	
	// Transaction support
	BeginTx(ctx context.Context) (*sqlx.Tx, error)
	
	// Connection management
	Ping(ctx context.Context) error
	Close() error
	
	// Driver metadata
	DriverName() string
	DriverVersion() string
}