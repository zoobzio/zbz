package zlog

// Config defines universal configuration for all zlog providers
type Config struct {
	// Basic configuration
	Name        string   `yaml:"name" json:"name"`
	Level       LogLevel `yaml:"level" json:"level"`
	Format      string   `yaml:"format" json:"format"`           // "json", "console", "text"
	Development bool     `yaml:"development" json:"development"` // Enable development mode

	// Output configuration
	Console bool           `yaml:"console" json:"console"`                     // Enable console output
	Outputs []OutputConfig `yaml:"outputs,omitempty" json:"outputs,omitempty"` // Additional outputs

	// Performance settings
	BufferSize int    `yaml:"buffer_size,omitempty" json:"buffer_size,omitempty"`
	FlushLevel string `yaml:"flush_level,omitempty" json:"flush_level,omitempty"`

	// Sampling configuration (for high-volume scenarios)
	Sampling *SamplingConfig `yaml:"sampling,omitempty" json:"sampling,omitempty"`

	// Provider-specific extensions
	Extensions map[string]any `yaml:"extensions,omitempty" json:"extensions,omitempty"`
}

// OutputConfig defines configuration for a single log output destination
type OutputConfig struct {
	Type    string         `yaml:"type" json:"type"`                           // "console", "file", "syslog"
	Level   string         `yaml:"level,omitempty" json:"level,omitempty"`     // Override global level
	Format  string         `yaml:"format,omitempty" json:"format,omitempty"`   // Override global format
	Target  string         `yaml:"target,omitempty" json:"target,omitempty"`   // File path, syslog tag, etc.
	Options map[string]any `yaml:"options,omitempty" json:"options,omitempty"` // Output-specific options
}

// SamplingConfig configures log sampling for high-volume scenarios
type SamplingConfig struct {
	Initial    int `yaml:"initial,omitempty" json:"initial,omitempty"`       // Sample first N messages per second
	Thereafter int `yaml:"thereafter,omitempty" json:"thereafter,omitempty"` // Then 1 in N thereafter per second
}

// Depot configuration removed - external storage handled via io.Writer interface

// DefaultConfig returns sensible defaults for zlog configuration
func DefaultConfig() Config {
	return Config{
		Name:        "zlog",
		Level:       INFO,
		Format:      "json",
		Development: false,
		Console:     true,
		BufferSize:  1024,
		FlushLevel:  "error",
	}
}

// DevelopmentConfig returns configuration suitable for development
func DevelopmentConfig() Config {
	return Config{
		Name:        "zlog-dev",
		Level:       DEBUG,
		Format:      "console",
		Development: true,
		Console:     true,
		BufferSize:  512,
	}
}

// ProductionConfig returns configuration suitable for production
func ProductionConfig() Config {
	return Config{
		Name:        "zlog-prod",
		Level:       INFO,
		Format:      "json",
		Development: false,
		Console:     false, // Production usually goes to files/external storage
		BufferSize:  4096,
		FlushLevel:  "warn",
	}
}

// WithDepot method removed - use zlog.PipeToWriter(depotWriter) instead

// String returns string representation of the level for backward compatibility
func (c Config) LevelString() string {
	return c.Level.String()
}
