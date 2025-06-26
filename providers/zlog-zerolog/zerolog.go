package zerolog

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"

	"zbz/zlog"
)

// zerologProvider implements ZlogProvider interface using zerolog
type zerologProvider struct {
	logger zerolog.Logger
}

// NewZerologLogger creates a zerolog logger contract with type-safe native client access
// Returns a contract that can be registered as the global singleton or used independently
// Example:
//   contract := zerologprovider.NewZerologLogger(config)
//   contract.Register()  // Register as global singleton
//   zerologLogger := contract.Native()  // Get *zerolog.Logger without casting
func NewZerologLogger(config zlog.ZlogConfig) (*zlog.ZlogContract[*zerolog.Logger], error) {
	// Apply defaults
	if config.Level == "" {
		config.Level = "info"
	}
	if config.Format == "" {
		config.Format = "json"
	}

	// Set global zerolog level
	zerolog.SetGlobalLevel(parseLogLevel(config.Level))

	// Configure time format
	zerolog.TimeFieldFormat = time.RFC3339

	var writers []io.Writer

	// If no outputs specified, default to console only
	// File outputs will be ignored unless hodor contract is set later
	if len(config.Outputs) == 0 {
		config.Outputs = []zlog.OutputConfig{
			{Type: "console", Level: config.Level, Format: config.Format},
		}
	}

	// Create writers for console outputs only initially
	for _, output := range config.Outputs {
		if output.Type == "console" {
			writer := createWriter(output, config.Format)
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

	// Combine writers
	var finalWriter io.Writer
	if len(writers) == 1 {
		finalWriter = writers[0]
	} else {
		finalWriter = zerolog.MultiLevelWriter(writers...)
	}

	// Create logger with combined writer
	logger := zerolog.New(finalWriter).With().Timestamp().Caller().Logger()

	// Apply sampling if configured
	if config.Sampling != nil {
		logger = logger.Sample(&zerolog.BasicSampler{N: uint32(config.Sampling.Thereafter)})
	}

	// Create provider wrapper
	provider := &zerologProvider{
		logger: logger,
	}

	// Create and return contract
	return zlog.NewContract[*zerolog.Logger](config.Name, provider, &logger, config), nil
}

// Info logs at info level
func (z *zerologProvider) Info(msg string, fields []zlog.Field) {
	event := z.logger.Info()
	z.addFieldsToEvent(event, fields)
	event.Msg(msg)
}

// Error logs at error level  
func (z *zerologProvider) Error(msg string, fields []zlog.Field) {
	event := z.logger.Error()
	z.addFieldsToEvent(event, fields)
	event.Msg(msg)
}

// Debug logs at debug level
func (z *zerologProvider) Debug(msg string, fields []zlog.Field) {
	event := z.logger.Debug()
	z.addFieldsToEvent(event, fields)
	event.Msg(msg)
}

// Warn logs at warn level
func (z *zerologProvider) Warn(msg string, fields []zlog.Field) {
	event := z.logger.Warn()
	z.addFieldsToEvent(event, fields)
	event.Msg(msg)
}

// Fatal logs at fatal level and exits
func (z *zerologProvider) Fatal(msg string, fields []zlog.Field) {
	event := z.logger.Fatal()
	z.addFieldsToEvent(event, fields)
	event.Msg(msg)
	// zerolog.Fatal() calls os.Exit(1) automatically
}

// Close cleans up the driver
func (z *zerologProvider) Close() error {
	// zerolog doesn't require explicit cleanup
	return nil
}


// addFieldsToEvent adds zlog fields to a zerolog event using the fluent API
func (z *zerologProvider) addFieldsToEvent(event *zerolog.Event, fields []zlog.Field) {
	for _, field := range fields {
		switch field.Type {
		case zlog.StringType:
			event.Str(field.Key, field.Value.(string))
		case zlog.IntType:
			event.Int(field.Key, field.Value.(int))
		case zlog.Int64Type:
			event.Int64(field.Key, field.Value.(int64))
		case zlog.Float64Type:
			event.Float64(field.Key, field.Value.(float64))
		case zlog.BoolType:
			event.Bool(field.Key, field.Value.(bool))
		case zlog.ErrorType:
			event.Err(field.Value.(error))
		case zlog.DurationType:
			event.Dur(field.Key, field.Value.(time.Duration))
		case zlog.TimeType:
			event.Time(field.Key, field.Value.(time.Time))
		case zlog.ByteStringType:
			event.Str(field.Key, field.Value.(string))
		case zlog.AnyType:
			event.Interface(field.Key, field.Value)
		case zlog.StringsType:
			event.Strs(field.Key, field.Value.([]string))
		default:
			// Fallback for unknown types
			event.Interface(field.Key, field.Value)
		}
	}
}

// createWriter creates an io.Writer for a specific output configuration
func createWriter(output zlog.OutputConfig, globalFormat string) io.Writer {
	format := output.Format
	if format == "" {
		format = globalFormat
	}

	var baseWriter io.Writer

	switch output.Type {
	case "console":
		baseWriter = os.Stdout
		// For console output, use pretty printing if format is "console"
		if format == "console" {
			baseWriter = zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC3339,
				NoColor:    false,
			}
		}
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
		
		baseWriter = &lumberjack.Logger{
			Filename:   target,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		}
	default:
		// Unknown output type
		return nil
	}

	// Apply level filtering if needed
	if output.Level != "" {
		level := parseLogLevel(output.Level)
		return zerolog.LevelWriterProvider{
			Writer: baseWriter,
			Level:  level,
		}
	}

	return baseWriter
}

// parseLogLevel converts string level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

