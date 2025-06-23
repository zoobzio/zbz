package logger

import (
	"os"
	"strings"
	"sync"
	"time"
)

// Global logger instance - auto-initializes on first use
var Log Logger
var once sync.Once

// Field represents a typed key-value pair for structured logging
type Field struct {
	Key   string
	Type  FieldType
	Value any
}

// FieldType defines the type of a log field for type safety
type FieldType int

const (
	StringType FieldType = iota
	IntType
	Int64Type
	Float64Type
	BoolType
	ErrorType
	DurationType
	TimeType
	ByteStringType
	AnyType
	StringsType
)

// Logger defines the interface for structured logging
type Logger interface {
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
}

// Type-safe field constructors (zap-like API)
func String(key, value string) Field {
	return Field{Key: key, Type: StringType, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Type: IntType, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Type: Int64Type, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Type: Float64Type, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Type: BoolType, Value: value}
}

func Err(err error) Field {
	return Field{Key: "error", Type: ErrorType, Value: err}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Type: DurationType, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Type: TimeType, Value: value}
}

func ByteString(key string, value []byte) Field {
	return Field{Key: key, Type: ByteStringType, Value: string(value)}
}

func Any(key string, value any) Field {
	return Field{Key: key, Type: AnyType, Value: value}
}

func Strings(key string, value []string) Field {
	return Field{Key: key, Type: StringsType, Value: value}
}

// init() runs automatically when package is imported
func init() {
	Log = getDefaultLogger()
}

// getDefaultLogger creates a logger based on environment configuration
func getDefaultLogger() Logger {
	// Check environment variable first
	loggerType := os.Getenv("ZBZ_LOGGER")

	switch strings.ToLower(loggerType) {
	case "zap":
		return NewZapLogger()
	case "noop", "none":
		return NewNoOpLogger()
	case "simple":
		return NewSimpleLogger()
	default:
		// Default to zap if nothing specified
		return NewZapLogger()
	}
}

// SetGlobalLogger allows users to replace the global logger
func SetGlobalLogger(l Logger) {
	Log = l
}

// Convenience functions that use the global logger
func Info(msg string, fields ...Field) {
	Log.Info(msg, fields...)
}

func Error(msg string, fields ...Field) {
	Log.Error(msg, fields...)
}

func Debug(msg string, fields ...Field) {
	Log.Debug(msg, fields...)
}

func Warn(msg string, fields ...Field) {
	Log.Warn(msg, fields...)
}

func Fatal(msg string, fields ...Field) {
	Log.Fatal(msg, fields...)
}