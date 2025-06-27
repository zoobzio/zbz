package zlog

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	zapProvider "zbz/providers/zlog-zap"
	"zbz/zlog"
	"zbz/cmd/demos/utils"
)

// JsonDemo demonstrates zlog with zap provider using JSON format
func JsonDemo() {
	fmt.Println("🚀 ZBZ Framework zlog + zap JSON Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	utils.Think("Let's explore how zlog creates beautiful structured JSON logs with the zap provider...")
	utils.Pause()
	
	utils.Demonstrate("First, we'll initialize zap with production-ready JSON formatting")
	zapContract := zapProvider.New(zlog.Config{
		Name:        "json-demo",
		Level:       zlog.DEBUG,
		Format:      "json",
		Development: false, // Production JSON format
		Console:     true,
	})
	
	utils.ShortPause()
	utils.Explain("Now we'll set up transparent output piping to capture logs to a file")
	file, _ := os.OpenFile("/tmp/zlog-json-demo.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	zlog.Pipe(file)
	defer file.Close()
	
	utils.ShortPause()
	utils.Demonstrate("Time to see structured logging in action with rich typed fields")
	zlog.Info("Application started",
		zlog.String("version", "1.0.0"),
		zlog.String("environment", "production"),
		zlog.Int("port", 8080),
		zlog.Bool("debug", false),
		zlog.Duration("startup_time", 250*time.Millisecond))
	
	// Show what this log entry looks like as JSON
	logEntry := map[string]any{
		"level": "info",
		"msg": "Application started",
		"version": "1.0.0",
		"environment": "production", 
		"port": 8080,
		"debug": false,
		"startup_time": "250ms",
	}
	utils.ShowJSON(logEntry)
	
	utils.ShortPause()
	utils.Explain("Watch how zlog handles complex nested data structures seamlessly")
	metadata := map[string]any{
		"build_time": "2024-01-15T10:30:00Z",
		"git_hash":   "abc123def",
		"features":   []string{"auth", "crud", "docs"},
		"config": map[string]any{
			"database": "postgresql",
			"cache":    "redis",
		},
	}
	
	zlog.Info("Service configuration loaded",
		zlog.String("service", "api-server"),
		zlog.Any("metadata", metadata),
		zlog.Strings("modules", []string{"auth", "db", "http"}))
	
	// Show the complex nested structure
	complexLog := map[string]any{
		"level": "info",
		"msg": "Service configuration loaded",
		"service": "api-server",
		"metadata": metadata,
		"modules": []string{"auth", "db", "http"},
	}
	utils.ShowJSON(complexLog)
	
	utils.ShortPause()
	utils.Demonstrate("Error logging becomes incredibly powerful with structured context")
	err := fmt.Errorf("database connection failed: timeout after 30s")
	zlog.Error("Database error occurred",
		zlog.Err(err),
		zlog.String("database_host", "db.example.com"),
		zlog.Int("port", 5432),
		zlog.String("database_name", "app_prod"),
		zlog.Duration("timeout", 30*time.Second),
		zlog.Int("retry_count", 3))
	
	// Show error log structure
	errorLog := map[string]any{
		"level": "error",
		"msg": "Database error occurred",
		"error": "database connection failed: timeout after 30s",
		"database_host": "db.example.com",
		"port": 5432,
		"database_name": "app_prod",
		"timeout": "30s",
		"retry_count": 3,
	}
	utils.ShowJSON(errorLog)
	
	utils.ShortPause()
	utils.Explain("Different log levels help categorize information by importance")
	zlog.Debug("Debug information",
		zlog.String("function", "processRequest"),
		zlog.String("details", "Request validation passed"))
	
	zlog.Warn("Performance warning",
		zlog.String("operation", "database_query"),
		zlog.Duration("duration", 2500*time.Millisecond),
		zlog.String("threshold", "2000ms"))
	
	utils.ShortPause()
	utils.Demonstrate("For ultimate control, you can access the underlying zap logger directly")
	zapLogger := zapContract.Logger()
	zapLogger.Info("Direct zap usage",
		zap.String("note", "This bypasses zlog field processing"),
		zap.String("format", "pure JSON"))
	
	utils.Pause()
	utils.Observe("Beautiful! We've created a complete JSON logging pipeline with structured data")
	utils.ShortPause()
	utils.Explain("Check /tmp/zlog-json-demo.log to see the perfectly formatted JSON output")
}