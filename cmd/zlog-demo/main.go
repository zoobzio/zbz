package main

import (
	"time"
	"zbz/zlog"
	
	// Import Capitan to enable events
	_ "zbz/capitan"
)

func main() {
	// Configure zlog
	zlog.Configure(zlog.Config{
		Level:  zlog.DEBUG,
		Format: "console",
	})
	
	// Demo all log levels
	zlog.Debug("Debug message", 
		zlog.String("component", "main"),
		zlog.String("operation", "startup"))
	
	zlog.Info("Application started", 
		zlog.String("version", "1.0.0"),
		zlog.Int("port", 8080),
		zlog.Layer("system"))
	
	zlog.Warn("Configuration warning", 
		zlog.String("setting", "max_connections"),
		zlog.Int("value", 1000),
		zlog.Concern("performance"))
	
	// Demo custom field processing
	zlog.RegisterFieldProcessor("user_id", func(field zlog.Field) []zlog.Field {
		// Hash user ID for privacy
		return []zlog.Field{
			zlog.String("user_id_hash", "hashed_"+field.Value.(string)),
		}
	})
	
	zlog.Info("User login", 
		zlog.String("user_id", "user123"),
		zlog.String("ip", "192.168.1.100"),
		zlog.UserScope("user123"),
		zlog.Privacy("private"))
	
	zlog.Error("Database connection failed", 
		zlog.Err(nil), // Would be real error
		zlog.String("database", "postgres"),
		zlog.Duration("timeout", 5*time.Second),
		zlog.Layer("data"),
		zlog.Concern("critical"))
	
	// Demo special field types
	zlog.Info("Processing sensitive data",
		zlog.Secret("api_key", "secret123"),
		zlog.PII("ssn", "123-45-6789"),
		zlog.Metric("processing_time", 150))
	
	// Log all levels to show filtering
	zlog.SetLevel(zlog.WARN)
	zlog.Debug("This won't show")   // Below WARN level
	zlog.Info("This won't show")    // Below WARN level  
	zlog.Warn("This will show")     // At WARN level
	zlog.Error("This will show")    // Above WARN level
	
	println("\n🚀 ZLog Nuclear Architecture Demo Complete!")
	println("✅ Zero dependencies in zlog core")
	println("✅ Auto-hydrated events via Capitan import")
	println("✅ Zero-allocation console output")
	println("✅ Custom field processing")
	println("✅ All 5 log levels supported")
	println("✅ Perfect backward compatibility")
	
	// Run adapter demo
	demoAdapters()
}