package zap

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"zbz/zlog"
)

// zapProvider implements ZlogProvider interface using zap
type zapProvider struct {
	logger *zap.Logger
}

// New creates a new zap-based contract with the provided configuration
func New(config Config) *zlog.ZlogContract[*zap.Logger] {
	// Apply defaults
	if config.Level == "" {
		config.Level = "info"
	}
	if config.Format == "" {
		config.Format = "json"
	}

	// If no outputs specified, default to console only
	// File outputs will be ignored unless hodor contract is set later
	if len(config.Outputs) == 0 {
		config.Outputs = []OutputConfig{
			{Type: "console", Level: config.Level, Format: config.Format},
		}
	}

	// Create cores for console outputs only initially
	var cores []zapcore.Core
	for _, output := range config.Outputs {
		if output.Type == "console" {
			core := createZapCore(output, config.Level, config.Format)
			if core != nil {
				cores = append(cores, core)
			}
		}
		// Skip file outputs - they require hodor contract
	}

	// Ensure we have at least console output
	if len(cores) == 0 {
		consoleOutput := OutputConfig{
			Type:   "console",
			Level:  config.Level,
			Format: config.Format,
		}
		core := createZapCore(consoleOutput, config.Level, config.Format)
		if core != nil {
			cores = append(cores, core)
		}
	}

	// Combine all cores
	var finalCore zapcore.Core
	if len(cores) == 1 {
		finalCore = cores[0]
	} else {
		finalCore = zapcore.NewTee(cores...)
	}

	// Apply sampling if configured
	if config.Sampling != nil {
		finalCore = zapcore.NewSamplerWithOptions(
			finalCore,
			time.Second, // Sample per second
			config.Sampling.Initial,
			config.Sampling.Thereafter,
		)
	}

	zapLog := zap.New(finalCore,
		zap.AddCaller(),
		zap.AddCallerSkip(3), // Skip: zapProvider.Info() -> zZlogService.Info() -> zlog.Info() -> user code
		zap.AddStacktrace(zap.ErrorLevel))

	// Create provider
	provider := &zapProvider{
		logger: zapLog,
	}

	// Return contract with both service and typed logger
	return zlog.NewContract[*zap.Logger](config.Name, provider, zapLog)
}

// Info logs at info level
func (z *zapProvider) Info(msg string, fields []zlog.Field) {
	zapFields := z.convertFields(fields)
	z.logger.Info(msg, zapFields...)
}

// Error logs at error level
func (z *zapProvider) Error(msg string, fields []zlog.Field) {
	zapFields := z.convertFields(fields)
	z.logger.Error(msg, zapFields...)
}

// Debug logs at debug level
func (z *zapProvider) Debug(msg string, fields []zlog.Field) {
	zapFields := z.convertFields(fields)
	z.logger.Debug(msg, zapFields...)
}

// Warn logs at warn level
func (z *zapProvider) Warn(msg string, fields []zlog.Field) {
	zapFields := z.convertFields(fields)
	z.logger.Warn(msg, zapFields...)
}

// Fatal logs at fatal level and exits
func (z *zapProvider) Fatal(msg string, fields []zlog.Field) {
	zapFields := z.convertFields(fields)
	z.logger.Fatal(msg, zapFields...)
}

// Close cleans up the provider
func (z *zapProvider) Close() error {
	return z.logger.Sync()
}




// convertFields converts zlog fields to zap fields
func (z *zapProvider) convertFields(fields []zlog.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = z.convertField(f)
	}
	return zapFields
}

// convertField converts a single zlog field to zap field
func (z *zapProvider) convertField(f zlog.Field) zap.Field {
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
		// Fallback to Any for unknown types (including pipeline types that shouldn't reach here)
		return zap.Any(f.Key, f.Value)
	}
}

// createZapCore creates a zapcore.Core for a specific output configuration
func createZapCore(output OutputConfig, globalLevel, globalFormat string) zapcore.Core {
	// Determine level for this output
	level := output.Level
	if level == "" {
		level = globalLevel
	}
	zapLevel := parseLogLevel(level)

	// Determine format for this output
	format := output.Format
	if format == "" {
		format = globalFormat
	}
	encoder := createEncoder(format)

	// Create writer syncer based on output type
	var syncer zapcore.WriteSyncer
	switch output.Type {
	case "console":
		syncer = zapcore.AddSync(os.Stdout)
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
		
		syncer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   target,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		})
	default:
		// Unknown output type, skip
		return nil
	}

	return zapcore.NewCore(encoder, syncer, zapLevel)
}

// parseLogLevel converts string level to zapcore.Level
func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

// createEncoder creates a zapcore.Encoder based on format
func createEncoder(format string) zapcore.Encoder {
	switch format {
	case "console":
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		return zapcore.NewConsoleEncoder(config)
	case "json":
		return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	case "logfmt":
		// Use console encoder with specific config for logfmt-like output
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncodeDuration = zapcore.StringDurationEncoder
		config.EncodeCaller = zapcore.ShortCallerEncoder
		return zapcore.NewConsoleEncoder(config)
	default:
		// Default to JSON
		return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}
}

