package zlog

import (
	"fmt"
	"io"
	"os"
	"time"
)

// UniversalConfig standardizes configuration across all providers
type UniversalConfig struct {
	Name    string
	Level   LogLevel
	Format  string
	Console bool
	Hodor   *HodorConfig
}

// HodorConfig defines hodor-specific output configuration
type HodorConfig struct {
	Contract     HodorContract
	KeyPrefix    string
	Rotation     RotationStrategy
	Compression  bool
	BufferSize   int
	FlushInterval time.Duration
}

// LogLevel represents universal log levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "debug"
	case INFO:
		return "info"
	case WARN:
		return "warn"
	case ERROR:
		return "error"
	case FATAL:
		return "fatal"
	default:
		return "info"
	}
}

// RotationStrategy defines how log files are rotated
type RotationStrategy struct {
	Method   string        // "size", "time", "hybrid"
	MaxSize  int64         // bytes
	MaxAge   time.Duration // duration
	MaxFiles int           // count
}

// OutputManager handles universal output creation
type OutputManager struct{}

// NewOutputManager creates a new output manager
func NewOutputManager() *OutputManager {
	return &OutputManager{}
}

// CreateConsoleWriter creates a console writer with appropriate formatting
func (o *OutputManager) CreateConsoleWriter(format string) io.Writer {
	// For now, all console output goes to stdout
	// TODO: Add format-specific console writers (colored output, etc.)
	return os.Stdout
}

// CreateHodorTeeWriter creates a writer that outputs to both console and hodor
func (o *OutputManager) CreateHodorTeeWriter(console io.Writer, hodor *HodorConfig) io.Writer {
	if hodor == nil || hodor.Contract == nil {
		// No hodor contract - console only
		return console
	}

	return &HodorTeeWriter{
		console:       console,
		hodor:         hodor.Contract,
		keyPrefix:     hodor.KeyPrefix,
		rotation:      hodor.Rotation,
		compression:   hodor.Compression,
		bufferSize:    hodor.BufferSize,
		flushInterval: hodor.FlushInterval,
		buffer:        make([]byte, 0, hodor.BufferSize),
		lastFlush:     time.Now(),
	}
}

// HodorTeeWriter writes to both console and hodor storage
type HodorTeeWriter struct {
	console       io.Writer
	hodor         HodorContract
	keyPrefix     string
	rotation      RotationStrategy
	compression   bool
	bufferSize    int
	flushInterval time.Duration
	
	// Internal state
	buffer     []byte
	lastFlush  time.Time
	currentKey string
	totalSize  int64
}

// Write implements io.Writer interface
func (h *HodorTeeWriter) Write(p []byte) (n int, err error) {
	// Always write to console first (never fail logging)
	n, err = h.console.Write(p)
	if err != nil {
		return n, err
	}

	// Send copy to hodor (async to avoid blocking)
	go h.writeToHodor(p)
	
	return n, nil
}

// writeToHodor handles the hodor storage logic
func (h *HodorTeeWriter) writeToHodor(data []byte) {
	// Add to buffer
	h.buffer = append(h.buffer, data...)
	h.totalSize += int64(len(data))

	// Check if we should flush
	shouldFlush := false
	
	// Flush if buffer is full
	if len(h.buffer) >= h.bufferSize {
		shouldFlush = true
	}
	
	// Flush if time interval has passed
	if time.Since(h.lastFlush) >= h.flushInterval {
		shouldFlush = true
	}
	
	// Flush if rotation is needed
	if h.shouldRotate() {
		shouldFlush = true
	}

	if shouldFlush {
		h.flush()
	}
}

// shouldRotate determines if log rotation is needed
func (h *HodorTeeWriter) shouldRotate() bool {
	switch h.rotation.Method {
	case "size":
		return h.totalSize >= h.rotation.MaxSize
	case "time":
		// Rotate daily by default
		if h.rotation.MaxAge == 0 {
			h.rotation.MaxAge = 24 * time.Hour
		}
		return time.Since(h.lastFlush) >= h.rotation.MaxAge
	case "hybrid":
		return h.totalSize >= h.rotation.MaxSize || time.Since(h.lastFlush) >= h.rotation.MaxAge
	default:
		// Default to size-based rotation
		return h.totalSize >= h.rotation.MaxSize
	}
}

// flush writes buffered data to hodor storage
func (h *HodorTeeWriter) flush() {
	if len(h.buffer) == 0 {
		return
	}

	// Generate storage key
	key := h.generateKey()
	
	// Prepare data
	data := make([]byte, len(h.buffer))
	copy(data, h.buffer)
	
	// TODO: Apply compression if enabled
	if h.compression {
		// data = compress(data)
	}

	// Write to hodor storage
	if err := h.hodor.Set(key, data, 0); err != nil {
		// Log error but don't fail - hodor errors shouldn't break logging
		// TODO: Use a fallback logger for hodor errors
		fmt.Printf("Error writing logs to hodor storage: %v\n", err)
		return
	}

	// Update state
	h.currentKey = key
	h.buffer = h.buffer[:0] // Clear buffer
	h.lastFlush = time.Now()
	h.totalSize = 0
}

// generateKey creates a storage key based on rotation strategy
func (h *HodorTeeWriter) generateKey() string {
	now := time.Now()
	
	switch h.rotation.Method {
	case "time":
		// Time-based rotation - include timestamp in key
		return fmt.Sprintf("%s-%s.log", h.keyPrefix, now.Format("2006-01-02-15-04-05"))
	case "size":
		// Size-based rotation - include size counter
		return fmt.Sprintf("%s-%s-size.log", h.keyPrefix, now.Format("2006-01-02"))
	case "hybrid":
		// Hybrid rotation - include both
		return fmt.Sprintf("%s-%s.log", h.keyPrefix, now.Format("2006-01-02-15-04-05"))
	default:
		// Default rotation
		return fmt.Sprintf("%s-%s.log", h.keyPrefix, now.Format("2006-01-02-15-04"))
	}
}

// Close flushes any remaining data
func (h *HodorTeeWriter) Close() error {
	h.flush()
	return nil
}

// LevelManager handles universal level parsing and conversion
type LevelManager struct{}

// NewLevelManager creates a new level manager
func NewLevelManager() *LevelManager {
	return &LevelManager{}
}

// ParseLevel converts string level to universal LogLevel
func (l *LevelManager) ParseLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return INFO
	}
}

// StandardizeConfig converts any provider config to universal format
func StandardizeConfig(config interface{}) (*UniversalConfig, error) {
	// TODO: Implement reflection-based config conversion
	// For now, return a basic universal config
	return &UniversalConfig{
		Name:    "default",
		Level:   INFO,
		Format:  "json",
		Console: true,
		Hodor:   nil,
	}, nil
}