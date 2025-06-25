package hodor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zbz/hodor"
	"zbz/zlog"
)

// hodorLogger implements ZlogProvider using hodor contracts for storage
type hodorLogger struct {
	contract   *hodor.HodorContract
	config     HodorLogConfig
	buffer     []LogEntry
	bufferSize int
}

// HodorLogConfig configures the hodor logger adapter
type HodorLogConfig struct {
	KeyPrefix    string        `yaml:"key_prefix"`    // Prefix for log file keys (e.g., "logs/app")
	Format       string        `yaml:"format"`        // "json" or "text"
	BufferSize   int           `yaml:"buffer_size"`   // Number of entries to buffer before write
	FlushInterval time.Duration `yaml:"flush_interval"` // How often to flush buffer
	RotateDaily  bool          `yaml:"rotate_daily"`  // Create new file each day
	Compression  bool          `yaml:"compression"`   // Compress log entries
}

// LogEntry represents a single log entry for storage
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Fields    map[string]any    `json:"fields,omitempty"`
	Source    string            `json:"source,omitempty"`
}

// NewHodorLogger creates a new hodor-based logger
func NewHodorLogger(contract *hodor.HodorContract, config HodorLogConfig) zlog.ZlogProvider {
	// Set defaults
	if config.KeyPrefix == "" {
		config.KeyPrefix = "logs/app"
	}
	if config.Format == "" {
		config.Format = "json"
	}
	if config.BufferSize == 0 {
		config.BufferSize = 100
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}

	logger := &hodorLogger{
		contract:   contract,
		config:     config,
		buffer:     make([]LogEntry, 0, config.BufferSize),
		bufferSize: config.BufferSize,
	}

	// Start background flusher
	go logger.backgroundFlusher()

	return logger
}

// Info logs an info message
func (h *hodorLogger) Info(msg string, fields []zlog.Field) {
	h.writeLog("INFO", msg, fields)
}

// Error logs an error message
func (h *hodorLogger) Error(msg string, fields []zlog.Field) {
	h.writeLog("ERROR", msg, fields)
}

// Debug logs a debug message
func (h *hodorLogger) Debug(msg string, fields []zlog.Field) {
	h.writeLog("DEBUG", msg, fields)
}

// Warn logs a warning message
func (h *hodorLogger) Warn(msg string, fields []zlog.Field) {
	h.writeLog("WARN", msg, fields)
}

// Fatal logs a fatal message
func (h *hodorLogger) Fatal(msg string, fields []zlog.Field) {
	h.writeLog("FATAL", msg, fields)
	h.flush() // Ensure fatal logs are written immediately
}

// writeLog creates a log entry and adds it to buffer
func (h *hodorLogger) writeLog(level, msg string, fields []zlog.Field) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    h.fieldsToMap(fields),
	}

	// Add to buffer
	h.buffer = append(h.buffer, entry)

	// Flush if buffer is full
	if len(h.buffer) >= h.bufferSize {
		h.flush()
	}
}

// fieldsToMap converts zlog fields to map for JSON storage
func (h *hodorLogger) fieldsToMap(fields []zlog.Field) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	result := make(map[string]any)
	for _, field := range fields {
		result[field.Key] = field.Value
	}
	return result
}

// flush writes buffered entries to hodor storage
func (h *hodorLogger) flush() {
	if len(h.buffer) == 0 {
		return
	}

	// Generate storage key
	key := h.generateKey()

	// Serialize entries
	var content []byte
	var err error

	if h.config.Format == "json" {
		content, err = json.Marshal(h.buffer)
	} else {
		content = h.formatAsText()
	}

	if err != nil {
		// In a real implementation, this would go to a fallback logger
		fmt.Printf("Error serializing logs: %v\n", err)
		return
	}

	// Write to hodor storage
	if err := h.contract.Set(key, content, 0); err != nil {
		// In a real implementation, this would go to a fallback logger
		fmt.Printf("Error writing logs to storage: %v\n", err)
		return
	}

	// Clear buffer
	h.buffer = h.buffer[:0]
}

// generateKey creates a storage key for logs
func (h *hodorLogger) generateKey() string {
	now := time.Now()
	
	if h.config.RotateDaily {
		return fmt.Sprintf("%s-%s.%s", 
			h.config.KeyPrefix,
			now.Format("2006-01-02"),
			h.getFileExtension())
	}
	
	return fmt.Sprintf("%s-%s.%s", 
		h.config.KeyPrefix,
		now.Format("2006-01-02-15-04"),
		h.getFileExtension())
}

// getFileExtension returns appropriate file extension
func (h *hodorLogger) getFileExtension() string {
	if h.config.Format == "json" {
		if h.config.Compression {
			return "json.gz"
		}
		return "json"
	}
	
	if h.config.Compression {
		return "log.gz"
	}
	return "log"
}

// formatAsText converts entries to text format
func (h *hodorLogger) formatAsText() []byte {
	var lines []string
	
	for _, entry := range h.buffer {
		line := fmt.Sprintf("%s [%s] %s", 
			entry.Timestamp.Format("2006-01-02T15:04:05.000Z"),
			entry.Level,
			entry.Message)
		
		// Add fields if present
		if len(entry.Fields) > 0 {
			var fieldStrs []string
			for key, value := range entry.Fields {
				fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", key, value))
			}
			line += " " + strings.Join(fieldStrs, " ")
		}
		
		lines = append(lines, line)
	}
	
	return []byte(strings.Join(lines, "\n") + "\n")
}

// backgroundFlusher periodically flushes the buffer
func (h *hodorLogger) backgroundFlusher() {
	ticker := time.NewTicker(h.config.FlushInterval)
	defer ticker.Stop()

	for range ticker.C {
		h.flush()
	}
}

// Close flushes any remaining logs and cleans up
func (h *hodorLogger) Close() error {
	h.flush()
	return nil
}

// NewContract creates a zlog contract using hodor storage
func NewContract(hodorContract *hodor.HodorContract, config HodorLogConfig) *zlog.ZlogContract[*hodorLogger] {
	logger := NewHodorLogger(hodorContract, config).(*hodorLogger)
	return zlog.NewContract("hodor-"+hodorContract.Name(), logger, logger)
}

// NewWithHodor creates a new zlog contract with hodor storage support
func NewWithHodor(name string, hodorContract *hodor.HodorContract, config HodorLogConfig) *zlog.ZlogContract[*hodorLogger] {
	if hodorContract == nil {
		// Return error - hodor adapter requires a hodor contract
		panic("hodor adapter requires a valid hodor contract")
	}
	
	logger := NewHodorLogger(hodorContract, config).(*hodorLogger)
	return zlog.NewContract(name, logger, logger)
}