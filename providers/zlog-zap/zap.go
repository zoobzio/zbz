package zap

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"zbz/zlog"
)

// zapProvider implements the zlog.ZlogProvider interface using zap
type zapProvider struct {
	logger *zap.Logger
}

// New creates a new zap logger and returns a self-registering zlog contract
// This automatically becomes the active logger for all zlog-using services
func New(config zlog.Config) *zlog.Contract[*zap.Logger] {
	// Create zap logger from universal config
	zapLogger := createZapLogger(config)
	
	// Create provider wrapper
	provider := &zapProvider{
		logger: zapLogger,
	}
	
	// Create and return self-registering contract
	return zlog.NewContract(config, zapLogger, provider)
}

// NewDevelopment creates a development zap logger with sensible defaults
func NewDevelopment() *zlog.Contract[*zap.Logger] {
	return New(zlog.DevelopmentConfig())
}

// NewProduction creates a production zap logger with sensible defaults  
func NewProduction() *zlog.Contract[*zap.Logger] {
	return New(zlog.ProductionConfig())
}

// createZapLogger builds a zap logger from Config
func createZapLogger(config zlog.Config) *zap.Logger {
	// Convert universal level to zap level
	var level zapcore.Level
	switch config.Level {
	case zlog.DEBUG:
		level = zapcore.DebugLevel
	case zlog.INFO:
		level = zapcore.InfoLevel
	case zlog.WARN:
		level = zapcore.WarnLevel
	case zlog.ERROR:
		level = zapcore.ErrorLevel
	case zlog.FATAL:
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}
	
	// Build zap config based on format
	var zapConfig zap.Config
	
	if config.Format == "console" || config.Console {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
	}
	
	// Apply universal settings
	zapConfig.Level = zap.NewAtomicLevelAt(level)
	
	// Set encoding based on format
	if config.Format == "console" {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapConfig.Encoding = "json"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}
	
	// Use transparent writer for all output - this enables piping
	transparentWriter := zlog.Writer()
	
	// Create core with transparent writer
	encoder := zapcore.NewJSONEncoder(zapConfig.EncoderConfig)
	if config.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
	}
	
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(transparentWriter),
		zapConfig.Level,
	)
	
	// Build logger with our transparent core
	logger := zap.New(core)
	
	// Add caller info if not disabled (check for development mode)
	if config.Console || config.Development {
		logger = logger.WithOptions(zap.AddCaller())
	}
	
	return logger
}

// No longer needed - we use Config directly

// ZlogProvider interface implementation

// Info logs an info message with structured fields
func (z *zapProvider) Info(msg string, fields []zlog.Field) {
	z.logger.Info(msg, convertFields(fields)...)
}

// Error logs an error message with structured fields
func (z *zapProvider) Error(msg string, fields []zlog.Field) {
	z.logger.Error(msg, convertFields(fields)...)
}

// Debug logs a debug message with structured fields
func (z *zapProvider) Debug(msg string, fields []zlog.Field) {
	z.logger.Debug(msg, convertFields(fields)...)
}

// Warn logs a warning message with structured fields
func (z *zapProvider) Warn(msg string, fields []zlog.Field) {
	z.logger.Warn(msg, convertFields(fields)...)
}

// Fatal logs a fatal message with structured fields and exits
func (z *zapProvider) Fatal(msg string, fields []zlog.Field) {
	z.logger.Fatal(msg, convertFields(fields)...)
}

// Close cleans up the zap logger
func (z *zapProvider) Close() error {
	return z.logger.Sync()
}

// convertFields converts zlog fields to zap fields
func convertFields(fields []zlog.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	
	for i, field := range fields {
		switch field.Type {
		case zlog.StringType:
			zapFields[i] = zap.String(field.Key, field.Value.(string))
		case zlog.IntType:
			zapFields[i] = zap.Int(field.Key, field.Value.(int))
		case zlog.Int64Type:
			zapFields[i] = zap.Int64(field.Key, field.Value.(int64))
		case zlog.Float64Type:
			zapFields[i] = zap.Float64(field.Key, field.Value.(float64))
		case zlog.BoolType:
			zapFields[i] = zap.Bool(field.Key, field.Value.(bool))
		case zlog.ErrorType:
			zapFields[i] = zap.Error(field.Value.(error))
		case zlog.DurationType:
			zapFields[i] = zap.Duration(field.Key, field.Value.(time.Duration))
		case zlog.TimeType:
			zapFields[i] = zap.Time(field.Key, field.Value.(time.Time))
		case zlog.ByteStringType:
			zapFields[i] = zap.ByteString(field.Key, []byte(field.Value.(string)))
		case zlog.StringsType:
			zapFields[i] = zap.Strings(field.Key, field.Value.([]string))
		case zlog.AnyType:
			zapFields[i] = zap.Any(field.Key, field.Value)
		default:
			// Fallback for unknown types
			zapFields[i] = zap.Any(field.Key, field.Value)
		}
	}
	
	return zapFields
}