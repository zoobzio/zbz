package zlog

import "io"

// Public API functions

// Info logs an info message with structured fields
func Info(msg string, fields ...Field) {
	processedFields := zlog.ProcessFields(fields)
	zlog.provider.Info(msg, processedFields)
}

// Error logs an error message with structured fields
func Error(msg string, fields ...Field) {
	processedFields := zlog.ProcessFields(fields)
	zlog.provider.Error(msg, processedFields)
}

// Debug logs a debug message with structured fields
func Debug(msg string, fields ...Field) {
	processedFields := zlog.ProcessFields(fields)
	zlog.provider.Debug(msg, processedFields)
}

// Warn logs a warning message with structured fields
func Warn(msg string, fields ...Field) {
	processedFields := zlog.ProcessFields(fields)
	zlog.provider.Warn(msg, processedFields)
}

// Fatal logs a fatal message with structured fields
func Fatal(msg string, fields ...Field) {
	processedFields := zlog.ProcessFields(fields)
	zlog.provider.Fatal(msg, processedFields)
}

// Process registers a field processor for a specific field type
func Process(fieldType FieldType, processor Processor) {
	zlog.Process(fieldType, processor)
}

// Pipe adds an io.Writer to receive copies of all log output
func Pipe(w io.Writer) {
	zlog.Pipe(w)
}

// Clear removes all registered pipe writers
func Clear() {
	zlog.ClearAllPipes()
}

// Writer returns the writer for providers to use
func Writer() io.Writer {
	return zlog.Writer()
}
