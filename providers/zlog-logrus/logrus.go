package logrus

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"zbz/zlog"
)

// logrusProvider implements ZlogProvider interface using Sirupsen Logrus
type logrusProvider struct {
	logger *logrus.Logger
}

// New creates a new logrus-based contract with the provided configuration
func New(config Config) *zlog.ZlogContract[*logrus.Logger] {
	// Apply defaults
	if config.Level == "" {
		config.Level = "info"
	}
	if config.Format == "" {
		config.Format = "json"
	}

	// Create new logrus logger
	logger := logrus.New()
	
	// Set global level
	level := parseLogrusLevel(config.Level)
	logger.SetLevel(level)

	// Configure formatter based on global format
	switch config.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "time",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "caller",
			},
		})
	case "text", "console":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "15:04:05",
			ForceColors:     true,
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// If no outputs specified, default to console only
	// File outputs will be ignored unless hodor contract is set later
	if len(config.Outputs) == 0 {
		config.Outputs = []OutputConfig{
			{Type: "console", Level: config.Level, Format: config.Format},
		}
	}

	// Set up console outputs only initially
	var writers []io.Writer
	for _, output := range config.Outputs {
		if output.Type == "console" {
			writer := createLogrusWriter(output)
			if writer != nil {
				writers = append(writers, writer)
			}
		}
		// Skip file outputs - they require hodor contract
	}

	// Ensure we have at least console output
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	// Set output
	if len(writers) == 1 {
		logger.SetOutput(writers[0])
	} else {
		logger.SetOutput(io.MultiWriter(writers...))
	}

	// Configure caller reporting
	logger.SetReportCaller(true)

	// TODO: Add hooks for advanced features like sampling
	// Logrus hooks could be used for:
	// - Sampling (drop logs based on rate)
	// - Metrics (send to prometheus)
	// - Alerting (send errors to slack)
	// - Correlation (add trace IDs)

	// Create provider
	provider := &logrusProvider{
		logger: logger,
	}

	// Return contract with both service and typed logger
	return zlog.NewContract[*logrus.Logger](config.Name, provider, logger)
}

// Info logs at info level
func (l *logrusProvider) Info(msg string, fields []zlog.Field) {
	entry := l.logger.WithFields(l.convertFields(fields))
	entry.Info(msg)
}

// Error logs at error level
func (l *logrusProvider) Error(msg string, fields []zlog.Field) {
	entry := l.logger.WithFields(l.convertFields(fields))
	entry.Error(msg)
}

// Debug logs at debug level
func (l *logrusProvider) Debug(msg string, fields []zlog.Field) {
	entry := l.logger.WithFields(l.convertFields(fields))
	entry.Debug(msg)
}

// Warn logs at warn level
func (l *logrusProvider) Warn(msg string, fields []zlog.Field) {
	entry := l.logger.WithFields(l.convertFields(fields))
	entry.Warn(msg)
}

// Fatal logs at fatal level and exits
func (l *logrusProvider) Fatal(msg string, fields []zlog.Field) {
	entry := l.logger.WithFields(l.convertFields(fields))
	entry.Fatal(msg)
}

// Close cleans up the driver
func (l *logrusProvider) Close() error {
	// Logrus doesn't require explicit cleanup
	return nil
}


// convertFields converts zlog fields to logrus fields
func (l *logrusProvider) convertFields(fields []zlog.Field) logrus.Fields {
	logrusFields := make(logrus.Fields, len(fields))
	
	for _, field := range fields {
		switch field.Type {
		case zlog.StringType:
			logrusFields[field.Key] = field.Value.(string)
		case zlog.IntType:
			logrusFields[field.Key] = field.Value.(int)
		case zlog.Int64Type:
			logrusFields[field.Key] = field.Value.(int64)
		case zlog.Float64Type:
			logrusFields[field.Key] = field.Value.(float64)
		case zlog.BoolType:
			logrusFields[field.Key] = field.Value.(bool)
		case zlog.ErrorType:
			// Logrus has special handling for errors
			if field.Key == "error" {
				logrusFields[logrus.ErrorKey] = field.Value.(error)
			} else {
				logrusFields[field.Key] = field.Value.(error).Error()
			}
		case zlog.DurationType:
			duration := field.Value.(time.Duration)
			logrusFields[field.Key] = duration.String()
			// Also add raw milliseconds for easier querying
			logrusFields[field.Key+"_ms"] = duration.Milliseconds()
		case zlog.TimeType:
			logrusFields[field.Key] = field.Value.(time.Time)
		case zlog.ByteStringType:
			logrusFields[field.Key] = field.Value.(string)
		case zlog.AnyType:
			logrusFields[field.Key] = field.Value
		case zlog.StringsType:
			logrusFields[field.Key] = field.Value
		default:
			// Fallback for unknown types
			logrusFields[field.Key] = field.Value
		}
	}
	
	return logrusFields
}

// createLogrusWriter creates an io.Writer for a specific output configuration
func createLogrusWriter(output OutputConfig) io.Writer {
	switch output.Type {
	case "console":
		return os.Stdout
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
		
		return &lumberjack.Logger{
			Filename:   target,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		}
	default:
		return nil
	}
}

// parseLogrusLevel converts string level to logrus.Level
func parseLogrusLevel(level string) logrus.Level {
	switch level {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	case "trace":
		return logrus.TraceLevel
	default:
		return logrus.InfoLevel
	}
}

// Custom hook for sampling (example of logrus hooks system)
type samplingHook struct {
	rate int
	count int
}

func (h *samplingHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *samplingHook) Fire(entry *logrus.Entry) error {
	h.count++
	if h.count % h.rate != 0 {
		// Skip this log entry
		entry.Data = nil
		entry.Message = ""
	}
	return nil
}

