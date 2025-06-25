package zap

import (
	"zbz/hodor"
	"zbz/zlog"
)

// Example of the new simplified API
func ExampleUsage() {
	// Create hodor contract for storage
	hodorContract := &hodor.HodorContract{
		// ... hodor contract setup
	}

	// Simple, clean configuration
	config := Config{
		Name:   "my-app",
		Level:  "info",
		Format: "json",
		// No file paths, no rotation config, no complex output arrays!
	}

	// Single constructor - enforces hodor for file storage
	loggerContract := NewWithHodor(config, hodorContract)

	// Use the logger
	logger := loggerContract.Zlog()
	logger.Info("Application started", zlog.String("version", "1.0.0"))

	// Files automatically go to hodor with:
	// - Universal rotation strategy
	// - Consistent key naming
	// - Automatic buffering and compression
	// - No provider-specific configuration needed!
}

// No more console-only constructor - forces intentional storage decisions
// func New(config Config) // ← This doesn't exist anymore!
//
// Want console-only logging? Use the simple provider:
// zlog.Configure(zlog.NewSimpleProvider())