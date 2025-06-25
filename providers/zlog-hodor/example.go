package hodor

import (
	"time"

	"zbz/hodor"
	"zbz/zlog"
)

// ExampleUsage shows how to use zlog with hodor storage
func ExampleUsage() {
	// 1. Create hodor storage contract (could be S3, GCS, etc.)
	s3Contract := hodor.NewMemory(nil) // In real usage: hodor.NewS3(config)
	err := s3Contract.Register("app-logs")
	if err != nil {
		panic(err)
	}
	defer s3Contract.Unregister()

	// 2. Configure hodor logger
	config := HodorLogConfig{
		KeyPrefix:     "logs/myapp",
		Format:        "json",
		BufferSize:    50,
		FlushInterval: 5 * time.Second,
		RotateDaily:   true,
		Compression:   false,
	}

	// 3. Create zlog contract with hodor storage
	logContract := NewContract(s3Contract, config)

	// 4. Get zlog service (now backed by cloud storage!)
	logger := logContract.Zlog()

	// 5. Use zlog normally - logs automatically go to cloud storage
	logger.Info("Application started", 
		zlog.String("version", "1.0.0"),
		zlog.String("environment", "production"))

	logger.Error("Database connection failed",
		zlog.String("host", "db.example.com"),
		zlog.Int("port", 5432),
		zlog.Err(err))

	// 6. Logs are automatically:
	//    - Buffered for performance
	//    - Flushed periodically  
	//    - Stored in cloud storage with organized keys
	//    - Available for log aggregation systems

	// Example storage keys generated:
	// "logs/myapp-2023-12-07.json" (daily rotation)
	// Content: [{"timestamp":"2023-12-07T10:30:00Z","level":"INFO","message":"Application started",...}]
}

// AdvancedUsage shows enterprise logging setup
func AdvancedUsage() {
	// Use different storage backends for different log types
	
	// Application logs → S3
	appStorage := hodor.NewMemory(nil) // hodor.NewS3(s3Config)
	appStorage.Register("app-logs")
	appContract := NewContract(appStorage, HodorLogConfig{
		KeyPrefix:     "logs/application",
		Format:        "json",
		RotateDaily:   true,
		BufferSize:    100,
	})

	// Audit logs → Different bucket with stricter retention
	auditStorage := hodor.NewMemory(nil) // hodor.NewS3(auditS3Config)  
	auditStorage.Register("audit-logs")
	auditContract := NewContract(auditStorage, HodorLogConfig{
		KeyPrefix:     "audit/events",
		Format:        "json",
		RotateDaily:   false, // Hourly rotation for compliance
		BufferSize:    10,    // Immediate flushing for audit
		FlushInterval: 1 * time.Second,
	})

	// Use different loggers for different purposes
	appLogger := appContract.Zlog()
	auditLogger := auditContract.Zlog()

	// Application logging
	appLogger.Info("User logged in", zlog.String("user_id", "123"))
	
	// Audit logging (stricter, faster flushing)
	auditLogger.Info("Permission granted", 
		zlog.String("user_id", "123"),
		zlog.String("resource", "user_data"),
		zlog.String("action", "read"))
}