package logger

// noOpLogger implements Logger interface with no-op operations (for testing)
type noOpLogger struct{}

// NewNoOpLogger creates a logger that does nothing (useful for tests)
func NewNoOpLogger() Logger {
	return &noOpLogger{}
}

func (n *noOpLogger) Info(msg string, fields ...Field)  {}
func (n *noOpLogger) Error(msg string, fields ...Field) {}
func (n *noOpLogger) Debug(msg string, fields ...Field) {}
func (n *noOpLogger) Warn(msg string, fields ...Field)  {}
func (n *noOpLogger) Fatal(msg string, fields ...Field) {
	// Even no-op logger should probably exit on Fatal
	// But don't log anything
	panic("fatal log message: " + msg)
}