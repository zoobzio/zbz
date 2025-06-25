package zlog

import (
	"fmt"
	"io"
	"os"
	"time"
)

// FormatManager handles universal format creation and conversion
type FormatManager struct{}

// NewFormatManager creates a new format manager
func NewFormatManager() *FormatManager {
	return &FormatManager{}
}

// SupportedFormats returns the formats supported by all providers
func (f *FormatManager) SupportedFormats() []string {
	return []string{"json", "console", "text"}
}

// CreateConsoleWriter creates a formatted console writer
func (f *FormatManager) CreateConsoleWriter(format string) io.Writer {
	switch format {
	case "json":
		return os.Stdout // Raw JSON to stdout
	case "console":
		return &ConsoleFormattedWriter{Writer: os.Stdout}
	case "text":
		return &TextFormattedWriter{Writer: os.Stdout}
	default:
		return os.Stdout
	}
}

// ConsoleFormattedWriter provides human-readable console output
type ConsoleFormattedWriter struct {
	Writer io.Writer
}

// Write formats log entries for human-readable console output
func (c *ConsoleFormattedWriter) Write(p []byte) (n int, err error) {
	// TODO: Parse JSON and reformat for console
	// For now, pass through as-is
	return c.Writer.Write(p)
}

// TextFormattedWriter provides simple text format output
type TextFormattedWriter struct {
	Writer io.Writer
}

// Write formats log entries as simple text
func (t *TextFormattedWriter) Write(p []byte) (n int, err error) {
	// TODO: Parse JSON and reformat as text
	// For now, pass through as-is
	return t.Writer.Write(p)
}

// ProviderFormatConverter helps convert between universal format specs and provider-specific encoders
type ProviderFormatConverter struct{}

// NewProviderFormatConverter creates a new provider format converter
func NewProviderFormatConverter() *ProviderFormatConverter {
	return &ProviderFormatConverter{}
}

// GetProviderFormat maps universal format to provider-specific format string
func (p *ProviderFormatConverter) GetProviderFormat(universalFormat, provider string) string {
	switch provider {
	case "zap":
		return p.getZapFormat(universalFormat)
	case "zerolog":
		return p.getZerologFormat(universalFormat)
	case "logrus":
		return p.getLogrusFormat(universalFormat)
	case "slog":
		return p.getSlogFormat(universalFormat)
	case "apex":
		return p.getApexFormat(universalFormat)
	default:
		return universalFormat
	}
}

// getZapFormat maps universal format to zap format
func (p *ProviderFormatConverter) getZapFormat(format string) string {
	switch format {
	case "json":
		return "json"
	case "console":
		return "console"
	case "text":
		return "console" // Zap uses console encoder for text-like output
	default:
		return "json"
	}
}

// getZerologFormat maps universal format to zerolog format
func (p *ProviderFormatConverter) getZerologFormat(format string) string {
	switch format {
	case "json":
		return "json"
	case "console":
		return "console"
	case "text":
		return "console"
	default:
		return "json"
	}
}

// getLogrusFormat maps universal format to logrus format
func (p *ProviderFormatConverter) getLogrusFormat(format string) string {
	switch format {
	case "json":
		return "json"
	case "console":
		return "text"
	case "text":
		return "text"
	default:
		return "json"
	}
}

// getSlogFormat maps universal format to slog format
func (p *ProviderFormatConverter) getSlogFormat(format string) string {
	switch format {
	case "json":
		return "json"
	case "console":
		return "text"
	case "text":
		return "text"
	default:
		return "json"
	}
}

// getApexFormat maps universal format to apex format
func (p *ProviderFormatConverter) getApexFormat(format string) string {
	switch format {
	case "json":
		return "json"
	case "console":
		return "cli"
	case "text":
		return "text"
	default:
		return "json"
	}
}

// DefaultRotationConfig provides sensible defaults for log rotation
func DefaultRotationConfig() RotationStrategy {
	return RotationStrategy{
		Method:   "hybrid",
		MaxSize:  10 * 1024 * 1024, // 10MB
		MaxAge:   24 * time.Hour,    // 1 day
		MaxFiles: 7,                 // 7 files
	}
}

// DefaultHodorConfig provides sensible defaults for hodor configuration
func DefaultHodorConfig(contract HodorContract, keyPrefix string) *HodorConfig {
	if keyPrefix == "" {
		keyPrefix = "logs/app"
	}
	
	return &HodorConfig{
		Contract:      contract,
		KeyPrefix:     keyPrefix,
		Rotation:      DefaultRotationConfig(),
		Compression:   true,
		BufferSize:    4096,           // 4KB buffer
		FlushInterval: 5 * time.Second, // Flush every 5 seconds
	}
}

// CreateUniversalConfig creates a universal config from provider-specific config
func CreateUniversalConfig(name, level, format string, console bool, hodor HodorContract, keyPrefix string) *UniversalConfig {
	levelMgr := NewLevelManager()
	
	config := &UniversalConfig{
		Name:    name,
		Level:   levelMgr.ParseLevel(level),
		Format:  format,
		Console: console,
		Hodor:   nil,
	}
	
	// Add hodor config if contract provided
	if hodor != nil {
		config.Hodor = DefaultHodorConfig(hodor, keyPrefix)
	}
	
	return config
}

// ValidateFormat ensures the format is supported
func ValidateFormat(format string) error {
	formatMgr := NewFormatManager()
	supported := formatMgr.SupportedFormats()
	
	for _, f := range supported {
		if f == format {
			return nil
		}
	}
	
	return fmt.Errorf("unsupported format '%s', supported formats: %v", format, supported)
}