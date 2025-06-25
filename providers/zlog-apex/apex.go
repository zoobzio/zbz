package apex

import (
	"io"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/multi"
	"github.com/apex/log/handlers/text"
	"gopkg.in/natefinch/lumberjack.v2"

	"zbz/zlog"
)

// apexProvider implements ZlogProvider interface using Apex Log
type apexProvider struct {
	logger log.Interface
}

// New creates a new apex-based contract with the provided configuration
func New(config Config) *zlog.ZlogContract[log.Interface] {
	// Apply defaults
	if config.Level == "" {
		config.Level = "info"
	}
	if config.Format == "" {
		config.Format = "json"
	}

	// Create logger instance
	logger := &log.Logger{
		Level: parseApexLevel(config.Level),
	}

	// If no outputs specified, default to console only
	// File outputs will be ignored unless hodor contract is set later
	if len(config.Outputs) == 0 {
		config.Outputs = []OutputConfig{
			{Type: "console", Level: config.Level, Format: config.Format},
		}
	}

	// Create handlers for console outputs only initially
	var handlers []log.Handler
	for _, output := range config.Outputs {
		if output.Type == "console" {
			handler := createApexHandler(output, config.Format)
			if handler != nil {
				handlers = append(handlers, handler)
			}
		}
		// Skip file outputs - they require hodor contract
	}

	// Set handler(s)
	if len(handlers) == 1 {
		logger.Handler = handlers[0]
	} else if len(handlers) > 1 {
		logger.Handler = multi.New(handlers...)
	} else {
		// Fallback to CLI handler
		logger.Handler = cli.New(os.Stdout)
	}

	// Create provider
	provider := &apexProvider{
		logger: logger,
	}

	// Return contract with both service and typed logger
	return zlog.NewContract[log.Interface](config.Name, provider, logger)
}

// Info logs at info level
func (a *apexProvider) Info(msg string, fields []zlog.Field) {
	entry := a.logger.WithFields(a.convertFields(fields))
	entry.Info(msg)
}

// Error logs at error level
func (a *apexProvider) Error(msg string, fields []zlog.Field) {
	entry := a.logger.WithFields(a.convertFields(fields))
	entry.Error(msg)
}

// Debug logs at debug level
func (a *apexProvider) Debug(msg string, fields []zlog.Field) {
	entry := a.logger.WithFields(a.convertFields(fields))
	entry.Debug(msg)
}

// Warn logs at warn level
func (a *apexProvider) Warn(msg string, fields []zlog.Field) {
	entry := a.logger.WithFields(a.convertFields(fields))
	entry.Warn(msg)
}

// Fatal logs at fatal level and exits
func (a *apexProvider) Fatal(msg string, fields []zlog.Field) {
	entry := a.logger.WithFields(a.convertFields(fields))
	entry.Fatal(msg)
}

// Close cleans up the driver
func (a *apexProvider) Close() error {
	// Apex log doesn't require explicit cleanup
	return nil
}


// convertFields converts zlog fields to apex log fields
func (a *apexProvider) convertFields(fields []zlog.Field) log.Fields {
	apexFields := make(log.Fields, len(fields))
	
	for _, field := range fields {
		switch field.Type {
		case zlog.StringType:
			apexFields[field.Key] = field.Value.(string)
		case zlog.IntType:
			apexFields[field.Key] = field.Value.(int)
		case zlog.Int64Type:
			apexFields[field.Key] = field.Value.(int64)
		case zlog.Float64Type:
			apexFields[field.Key] = field.Value.(float64)
		case zlog.BoolType:
			apexFields[field.Key] = field.Value.(bool)
		case zlog.ErrorType:
			// Apex log handles errors as strings
			apexFields[field.Key] = field.Value.(error).Error()
		case zlog.DurationType:
			duration := field.Value.(time.Duration)
			apexFields[field.Key] = duration.String()
			// Also provide milliseconds for easier querying
			apexFields[field.Key+"_ms"] = duration.Milliseconds()
		case zlog.TimeType:
			apexFields[field.Key] = field.Value.(time.Time)
		case zlog.ByteStringType:
			apexFields[field.Key] = field.Value.(string)
		case zlog.AnyType:
			apexFields[field.Key] = field.Value
		case zlog.StringsType:
			apexFields[field.Key] = field.Value
		default:
			// Fallback for unknown types
			apexFields[field.Key] = field.Value
		}
	}
	
	return apexFields
}

// createApexHandler creates an apex log handler for a specific output configuration
func createApexHandler(output OutputConfig, globalFormat string) log.Handler {
	format := output.Format
	if format == "" {
		format = globalFormat
	}

	var writer io.Writer

	switch output.Type {
	case "console":
		writer = os.Stdout
	case "file":
		target := output.Target
		if target == "" {
			target = ".logs/app.log"
		}
		
		// Check for rotation options
		maxSize := 10
		maxBackups := 5
		maxAge := 7
		compress := true
		
		if options := output.Options; options != nil {
			if ms, ok := options["max_size"].(int); ok {
				maxSize = ms
			}
			if mb, ok := options["max_backups"].(int); ok {
				maxBackups = mb
			}
			if ma, ok := options["max_age"].(int); ok {
				maxAge = ma
			}
			if c, ok := options["compress"].(bool); ok {
				compress = c
			}
		}
		
		writer = &lumberjack.Logger{
			Filename:   target,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		}
	default:
		return nil
	}

	// Create appropriate handler based on format
	switch format {
	case "json":
		return json.New(writer)
	case "text":
		return text.New(writer)
	case "cli", "console":
		return cli.New(writer)
	default:
		return json.New(writer)
	}
}

// parseApexLevel converts string level to apex log level
func parseApexLevel(level string) log.Level {
	switch level {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	default:
		return log.InfoLevel
	}
}

