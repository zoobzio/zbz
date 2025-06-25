package slog

// Config defines the configuration for the slog provider
type Config struct {
	Name    string         `json:"name"`
	Level   string         `json:"level,omitempty"`    // "debug", "info", "warn", "error"
	Format  string         `json:"format,omitempty"`   // "json", "text"
	Outputs []OutputConfig `json:"outputs,omitempty"`  // Multiple output destinations
}

// OutputConfig defines configuration for a single log output destination
type OutputConfig struct {
	Type    string         `json:"type"`               // "console", "file"
	Level   string         `json:"level,omitempty"`    // Override global level for this output
	Format  string         `json:"format,omitempty"`   // Override global format for this output
	Target  string         `json:"target,omitempty"`   // File path for file outputs
	Options map[string]any `json:"options,omitempty"`  // Output-specific options (rotation, etc.)
}