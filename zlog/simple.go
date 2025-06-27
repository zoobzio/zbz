package zlog

import (
	"fmt"
	"strings"
	"time"
)

// simpleProvider implements Provider interface with minimal, readable console output
type simpleProvider struct {
	minLevel logLevel
}

type logLevel int

const (
	levelDebug logLevel = iota
	levelInfo
	levelWarn
	levelError
	levelFatal
)

// newSimpleProvider creates a new simple console provider
func newSimpleProvider() Provider {
	return &simpleProvider{
		minLevel: levelInfo, // Default to info level
	}
}

// Info logs at info level
func (s *simpleProvider) Info(msg string, fields []Field) {
	s.log(levelInfo, "INFO", msg, fields)
}

// Error logs at error level
func (s *simpleProvider) Error(msg string, fields []Field) {
	s.log(levelError, "ERROR", msg, fields)
}

// Debug logs at debug level
func (s *simpleProvider) Debug(msg string, fields []Field) {
	s.log(levelDebug, "DEBUG", msg, fields)
}

// Warn logs at warn level
func (s *simpleProvider) Warn(msg string, fields []Field) {
	s.log(levelWarn, "WARN", msg, fields)
}

// Fatal logs at fatal level and exits
func (s *simpleProvider) Fatal(msg string, fields []Field) {
	s.log(levelFatal, "FATAL", msg, fields)
	// Don't exit in tests - services can override if they want different behavior
}

// Close cleans up the driver
func (s *simpleProvider) Close() error {
	return nil
}


// log is the core logging function
func (s *simpleProvider) log(level logLevel, levelStr, msg string, fields []Field) {
	// Skip if below minimum level
	if level < s.minLevel {
		return
	}

	// Format: 15:04:05 INFO  message key=value key2=value2
	timestamp := time.Now().Format("15:04:05")
	
	var parts []string
	parts = append(parts, timestamp)
	parts = append(parts, fmt.Sprintf("%-5s", levelStr))
	parts = append(parts, msg)
	
	// Add fields as key=value pairs
	for _, field := range fields {
		fieldStr := s.formatField(field)
		if fieldStr != "" {
			parts = append(parts, fieldStr)
		}
	}
	
	// Print to the service's writer (which handles piping)
	output := strings.Join(parts, " ") + "\n"
	
	// Get the service writer that handles piping
	writer := zlog.Writer()
	writer.Write([]byte(output))
}

// formatField converts a zlog field to a simple key=value string
func (s *simpleProvider) formatField(field Field) string {
	switch field.Type {
	case StringType:
		value := field.Value.(string)
		// Quote strings with spaces
		if strings.Contains(value, " ") {
			return fmt.Sprintf(`%s="%s"`, field.Key, value)
		}
		return fmt.Sprintf("%s=%s", field.Key, value)
	case IntType:
		return fmt.Sprintf("%s=%d", field.Key, field.Value.(int))
	case Int64Type:
		return fmt.Sprintf("%s=%d", field.Key, field.Value.(int64))
	case Float64Type:
		return fmt.Sprintf("%s=%.2f", field.Key, field.Value.(float64))
	case BoolType:
		return fmt.Sprintf("%s=%t", field.Key, field.Value.(bool))
	case ErrorType:
		err := field.Value.(error)
		return fmt.Sprintf(`%s="%s"`, field.Key, err.Error())
	case DurationType:
		duration := field.Value.(time.Duration)
		return fmt.Sprintf("%s=%s", field.Key, duration.String())
	case TimeType:
		t := field.Value.(time.Time)
		return fmt.Sprintf("%s=%s", field.Key, t.Format(time.RFC3339))
	case ByteStringType:
		value := field.Value.(string)
		if strings.Contains(value, " ") {
			return fmt.Sprintf(`%s="%s"`, field.Key, value)
		}
		return fmt.Sprintf("%s=%s", field.Key, value)
	case StringsType:
		strs := field.Value.([]string)
		joined := strings.Join(strs, ",")
		return fmt.Sprintf(`%s="[%s]"`, field.Key, joined)
	case AnyType:
		return fmt.Sprintf(`%s="%v"`, field.Key, field.Value)
	default:
		return fmt.Sprintf(`%s="%v"`, field.Key, field.Value)
	}
}