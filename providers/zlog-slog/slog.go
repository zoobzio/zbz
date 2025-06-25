package slog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"zbz/zlog"
)

// slogProvider implements ZlogProvider interface using Go's standard slog
type slogProvider struct {
	logger *slog.Logger
}

// New creates a new slog-based contract with the provided configuration
func New(config Config) *zlog.ZlogContract[*slog.Logger] {
	// Apply defaults
	if config.Level == "" {
		config.Level = "info"
	}
	if config.Format == "" {
		config.Format = "json"
	}

	var handlers []slog.Handler

	// If no outputs specified, default to console only
	// File outputs will be ignored unless hodor contract is set later
	if len(config.Outputs) == 0 {
		config.Outputs = []OutputConfig{
			{Type: "console", Level: config.Level, Format: config.Format},
		}
	}

	// Create handlers for console outputs only initially
	for _, output := range config.Outputs {
		if output.Type == "console" {
			handler := createSlogHandler(output, config.Format)
			if handler != nil {
				handlers = append(handlers, handler)
			}
		}
		// Skip file outputs - they require hodor contract
	}

	// Ensure we have at least console output
	if len(handlers) == 0 {
		handlers = append(handlers, slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	// Combine handlers using a multi-handler
	var finalHandler slog.Handler
	if len(handlers) == 1 {
		finalHandler = handlers[0]
	} else {
		finalHandler = &multiHandler{handlers: handlers}
	}

	// Create logger with combined handler
	logger := slog.New(finalHandler)

	// TODO: Apply sampling if configured (slog doesn't have built-in sampling)

	// Create provider
	provider := &slogProvider{
		logger: logger,
	}

	// Return contract with both service and typed logger
	return zlog.NewContract[*slog.Logger](config.Name, provider, logger)
}

// Info logs at info level
func (s *slogProvider) Info(msg string, fields []zlog.Field) {
	attrs := s.convertFields(fields)
	s.logger.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}

// Error logs at error level
func (s *slogProvider) Error(msg string, fields []zlog.Field) {
	attrs := s.convertFields(fields)
	s.logger.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
}

// Debug logs at debug level
func (s *slogProvider) Debug(msg string, fields []zlog.Field) {
	attrs := s.convertFields(fields)
	s.logger.LogAttrs(context.Background(), slog.LevelDebug, msg, attrs...)
}

// Warn logs at warn level
func (s *slogProvider) Warn(msg string, fields []zlog.Field) {
	attrs := s.convertFields(fields)
	s.logger.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
}

// Fatal logs at fatal level and exits (slog doesn't have Fatal, so we use Error + exit)
func (s *slogProvider) Fatal(msg string, fields []zlog.Field) {
	attrs := s.convertFields(fields)
	s.logger.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
	os.Exit(1)
}

// Close cleans up the driver
func (s *slogProvider) Close() error {
	// slog doesn't require explicit cleanup
	return nil
}


// convertFields converts zlog fields to slog attributes
func (s *slogProvider) convertFields(fields []zlog.Field) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(fields))
	
	for _, field := range fields {
		switch field.Type {
		case zlog.StringType:
			attrs = append(attrs, slog.String(field.Key, field.Value.(string)))
		case zlog.IntType:
			attrs = append(attrs, slog.Int(field.Key, field.Value.(int)))
		case zlog.Int64Type:
			attrs = append(attrs, slog.Int64(field.Key, field.Value.(int64)))
		case zlog.Float64Type:
			attrs = append(attrs, slog.Float64(field.Key, field.Value.(float64)))
		case zlog.BoolType:
			attrs = append(attrs, slog.Bool(field.Key, field.Value.(bool)))
		case zlog.ErrorType:
			// slog represents errors as strings
			attrs = append(attrs, slog.String("error", field.Value.(error).Error()))
		case zlog.DurationType:
			attrs = append(attrs, slog.Duration(field.Key, field.Value.(time.Duration)))
		case zlog.TimeType:
			attrs = append(attrs, slog.Time(field.Key, field.Value.(time.Time)))
		case zlog.ByteStringType:
			attrs = append(attrs, slog.String(field.Key, field.Value.(string)))
		case zlog.AnyType:
			attrs = append(attrs, slog.Any(field.Key, field.Value))
		case zlog.StringsType:
			// slog doesn't have a built-in string slice type, use Any
			attrs = append(attrs, slog.Any(field.Key, field.Value))
		default:
			// Fallback for unknown types
			attrs = append(attrs, slog.Any(field.Key, field.Value))
		}
	}
	
	return attrs
}

// createSlogHandler creates an slog.Handler for a specific output configuration
func createSlogHandler(output OutputConfig, globalFormat string) slog.Handler {
	format := output.Format
	if format == "" {
		format = globalFormat
	}

	level := parseSlogLevel(output.Level)

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

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: true, // Include caller info
	}

	// Create appropriate handler based on format
	switch format {
	case "json":
		return slog.NewJSONHandler(writer, opts)
	case "console", "text":
		return slog.NewTextHandler(writer, opts)
	default:
		return slog.NewJSONHandler(writer, opts)
	}
}

// parseSlogLevel converts string level to slog.Level
func parseSlogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// multiHandler implements slog.Handler to write to multiple handlers
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Enable if any handler is enabled for this level
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	// Send to all handlers
	for _, h := range m.handlers {
		if h.Enabled(ctx, record.Level) {
			// Clone the record for each handler since they might modify it
			cloned := record
			if err := h.Handle(ctx, cloned); err != nil {
				// Continue with other handlers even if one fails
				continue
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}

