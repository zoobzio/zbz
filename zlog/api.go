package zlog

import "os"

// Public API functions - Zero breaking changes

// Debug logs a debug message with structured fields
func Debug(msg string, fields ...Field) {
	if !zlog.shouldLog(DEBUG) {
		return
	}
	processedFields := zlog.processFields(fields)
	writeConsole("DEBUG", msg, processedFields)
	zlog.emitEvent("DEBUG", msg, processedFields)
}

// Info logs an info message with structured fields
func Info(msg string, fields ...Field) {
	if !zlog.shouldLog(INFO) {
		return
	}
	processedFields := zlog.processFields(fields)
	writeConsole("INFO", msg, processedFields)
	zlog.emitEvent("INFO", msg, processedFields)
}

// Warn logs a warning message with structured fields
func Warn(msg string, fields ...Field) {
	if !zlog.shouldLog(WARN) {
		return
	}
	processedFields := zlog.processFields(fields)
	writeConsole("WARN", msg, processedFields)
	zlog.emitEvent("WARN", msg, processedFields)
}

// Error logs an error message with structured fields
func Error(msg string, fields ...Field) {
	if !zlog.shouldLog(ERROR) {
		return
	}
	processedFields := zlog.processFields(fields)
	writeConsole("ERROR", msg, processedFields)
	zlog.emitEvent("ERROR", msg, processedFields)
}

// Fatal logs a fatal message with structured fields and exits
func Fatal(msg string, fields ...Field) {
	processedFields := zlog.processFields(fields)
	writeConsole("FATAL", msg, processedFields)
	zlog.emitEvent("FATAL", msg, processedFields)
	os.Exit(1)
}

// Configuration functions

// SetLevel sets the minimum log level
func SetLevel(level LogLevel) {
	zlog.mu.Lock()
	defer zlog.mu.Unlock()
	zlog.level = level
}

// GetLevel returns the current log level
func GetLevel() LogLevel {
	zlog.mu.RLock()
	defer zlog.mu.RUnlock()
	return zlog.level
}
