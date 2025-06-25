package zlog

import "io"

// Private concrete logger instance
var zlog *zZlog

// Universal service instances
var (
	outputManager    *OutputManager
	levelManager     *LevelManager
	formatManager    *FormatManager
	formatConverter  *ProviderFormatConverter
)

// init sets up a default simple logger and universal services
func init() {
	// Initialize universal services
	outputManager = NewOutputManager()
	levelManager = NewLevelManager()
	formatManager = NewFormatManager()
	formatConverter = NewProviderFormatConverter()
	
	// Create a simple console logger as the default
	// This can be replaced by any service that needs custom logging
	provider := newSimpleProvider()
	zlog = NewZlogService(provider)
}

// Public API functions

// Info logs an info message with structured fields
func Info(msg string, fields ...Field) {
	zlog.Info(msg, fields...)
}

// Error logs an error message with structured fields
func Error(msg string, fields ...Field) {
	zlog.Error(msg, fields...)
}

// Debug logs a debug message with structured fields
func Debug(msg string, fields ...Field) {
	zlog.Debug(msg, fields...)
}

// Warn logs a warning message with structured fields
func Warn(msg string, fields ...Field) {
	zlog.Warn(msg, fields...)
}

// Fatal logs a fatal message with structured fields
func Fatal(msg string, fields ...Field) {
	zlog.Fatal(msg, fields...)
}

// Configure replaces the current logger with a new provider
func Configure(provider ZlogProvider) {
	zlog = NewZlogService(provider)
	// Note: contract field will be set by the calling contract's Zlog() method
}

// Universal Service Access Functions

// GetOutputManager returns the universal output manager
func GetOutputManager() *OutputManager {
	return outputManager
}

// GetLevelManager returns the universal level manager
func GetLevelManager() *LevelManager {
	return levelManager
}

// GetFormatManager returns the universal format manager
func GetFormatManager() *FormatManager {
	return formatManager
}

// GetFormatConverter returns the universal format converter
func GetFormatConverter() *ProviderFormatConverter {
	return formatConverter
}

// CreateStandardWriter creates a standard writer for console + hodor output
func CreateStandardWriter(format string, hodorContract HodorContract, keyPrefix string) io.Writer {
	console := formatManager.CreateConsoleWriter(format)
	
	if hodorContract == nil {
		return console
	}
	
	hodorConfig := DefaultHodorConfig(hodorContract, keyPrefix)
	return outputManager.CreateHodorTeeWriter(console, hodorConfig)
}

// ParseLevel provides direct access to level parsing
func ParseLevel(level string) LogLevel {
	return levelManager.ParseLevel(level)
}

// ConvertLevel creates a level converter for provider-specific conversion
func ConvertLevel(level LogLevel) *LevelConverter {
	return NewLevelConverter(level)
}

// ZlogService defines the interface for structured logging
type ZlogService interface {
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
}
