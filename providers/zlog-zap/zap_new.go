package zap

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"zbz/zlog"
)

// zapProvider implements ZlogProvider interface using zap
type zapProvider struct {
	logger *zap.Logger
}

// NewZapLogger creates a zap logger contract with type-safe native client access
// Returns a contract that can be registered as the global singleton or used independently
// Example:
//   contract := zlogzap.NewZapLogger(config)
//   contract.Register()  // Register as global singleton
//   zapLogger := contract.Native()  // Get *zap.Logger without casting
func NewZapLogger(config zlog.ZlogConfig) (*zlog.ZlogContract[*zap.Logger], error) {
	// Apply defaults
	if config.Level == "" {
		config.Level = "info"
	}
	if config.Format == "" {
		config.Format = "json"
	}
	if config.Name == "" {
		config.Name = "zap-logger"
	}

	// Parse level
	zapLevel, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}

	// Create encoder config
	var encoderConfig zapcore.EncoderConfig
	if config.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
	}

	// Create encoder based on format
	var encoder zapcore.Encoder
	switch config.Format {
	case "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default: // "json"
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create output syncer
	var syncer zapcore.WriteSyncer
	if config.OutputFile != "" {
		// File output - simple implementation
		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		syncer = zapcore.AddSync(file)
	} else {
		// Console output
		syncer = zapcore.Lock(os.Stdout)
	}

	// Create core
	core := zapcore.NewCore(encoder, syncer, zapLevel)

	// Apply sampling if configured
	if config.Sampling != nil {
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second,
			config.Sampling.Initial,
			config.Sampling.Thereafter,
		)
	}

	// Create zap logger
	zapLogger := zap.New(core, 
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel))

	// Create provider wrapper
	provider := &zapProvider{
		logger: zapLogger,
	}

	// Create and return contract
	return zlog.NewContract[*zap.Logger](config.Name, provider, zapLogger, config), nil
}

// Info logs at info level
func (z *zapProvider) Info(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Info(msg, zapFields...)
}

// Error logs at error level
func (z *zapProvider) Error(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Error(msg, zapFields...)
}

// Debug logs at debug level
func (z *zapProvider) Debug(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Debug(msg, zapFields...)
}

// Warn logs at warn level
func (z *zapProvider) Warn(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Warn(msg, zapFields...)
}

// Fatal logs at fatal level and exits
func (z *zapProvider) Fatal(msg string, fields []zlog.Field) {
	zapFields := convertFields(fields)
	z.logger.Fatal(msg, zapFields...)
}

// Close cleans up the provider
func (z *zapProvider) Close() error {
	return z.logger.Sync()
}

// convertFields converts zlog fields to zap fields (simplified - could be universalized)
func convertFields(fields []zlog.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = convertField(f)
	}
	return zapFields
}

// convertField converts a single zlog field to zap field (could be universalized)
func convertField(f zlog.Field) zap.Field {
	switch f.Type {
	case zlog.StringType:
		return zap.String(f.Key, f.Value.(string))
	case zlog.IntType:
		return zap.Int(f.Key, f.Value.(int))
	case zlog.Int64Type:
		return zap.Int64(f.Key, f.Value.(int64))
	case zlog.Float64Type:
		return zap.Float64(f.Key, f.Value.(float64))
	case zlog.BoolType:
		return zap.Bool(f.Key, f.Value.(bool))
	case zlog.ErrorType:
		return zap.Error(f.Value.(error))
	case zlog.DurationType:
		return zap.Duration(f.Key, f.Value.(time.Duration))
	case zlog.TimeType:
		return zap.Time(f.Key, f.Value.(time.Time))
	case zlog.ByteStringType:
		return zap.String(f.Key, f.Value.(string))
	case zlog.AnyType:
		return zap.Any(f.Key, f.Value)
	case zlog.StringsType:
		return zap.Strings(f.Key, f.Value.([]string))
	default:
		return zap.Any(f.Key, f.Value)
	}
}

// createZapEncoder creates appropriate zap encoder (simplified)
func createZapEncoder(format string) zapcore.Encoder {
	switch format {
	case "console":
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		return zapcore.NewConsoleEncoder(config)
	case "json":
		return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	default:
		return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}
}