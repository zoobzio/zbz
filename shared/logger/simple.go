package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// simpleLogger implements Logger interface with basic stdout logging
type simpleLogger struct {
	logger *log.Logger
}

// NewSimpleLogger creates a basic stdout logger (useful for debugging)
func NewSimpleLogger() Logger {
	return &simpleLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}
}

func (s *simpleLogger) Info(msg string, fields ...Field) {
	s.logWithLevel("INFO", msg, fields...)
}

func (s *simpleLogger) Error(msg string, fields ...Field) {
	s.logWithLevel("ERROR", msg, fields...)
}

func (s *simpleLogger) Debug(msg string, fields ...Field) {
	s.logWithLevel("DEBUG", msg, fields...)
}

func (s *simpleLogger) Warn(msg string, fields ...Field) {
	s.logWithLevel("WARN", msg, fields...)
}

func (s *simpleLogger) Fatal(msg string, fields ...Field) {
	s.logWithLevel("FATAL", msg, fields...)
	os.Exit(1)
}

func (s *simpleLogger) logWithLevel(level, msg string, fields ...Field) {
	fieldStr := ""
	if len(fields) > 0 {
		fieldStr = " " + s.formatFields(fields)
	}
	s.logger.Printf("[%s] %s%s", level, msg, fieldStr)
}

func (s *simpleLogger) formatFields(fields []Field) string {
	if len(fields) == 0 {
		return ""
	}
	
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += " "
		}
		result += fmt.Sprintf("%s=%v", f.Key, s.formatValue(f))
	}
	return result
}

func (s *simpleLogger) formatValue(f Field) string {
	switch f.Type {
	case ErrorType:
		if err, ok := f.Value.(error); ok {
			return err.Error()
		}
		return fmt.Sprintf("%v", f.Value)
	case TimeType:
		if t, ok := f.Value.(time.Time); ok {
			return t.Format(time.RFC3339)
		}
		return fmt.Sprintf("%v", f.Value)
	case DurationType:
		if d, ok := f.Value.(time.Duration); ok {
			return d.String()
		}
		return fmt.Sprintf("%v", f.Value)
	default:
		return fmt.Sprintf("%v", f.Value)
	}
}