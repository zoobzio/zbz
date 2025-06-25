package zerolog

// Config defines the configuration for the zerolog provider
type Config struct {
	Name     string         `json:"name"`
	Level    string         `json:"level,omitempty"`    // "debug", "info", "warn", "error", "fatal", "trace"
	Format   string         `json:"format,omitempty"`   // "json", "console"
	Outputs  []OutputConfig `json:"outputs,omitempty"`  // Multiple output destinations
	Sampling *SamplingConfig `json:"sampling,omitempty"` // Rate limiting for high-volume logs
}

// OutputConfig defines configuration for a single log output destination
type OutputConfig struct {
	Type    string         `json:"type"`               // "console", "file"
	Level   string         `json:"level,omitempty"`    // Override global level for this output
	Format  string         `json:"format,omitempty"`   // Override global format for this output
	Target  string         `json:"target,omitempty"`   // File path for file outputs
	Options map[string]any `json:"options,omitempty"`  // Output-specific options (rotation, etc.)
}

// SamplingConfig configures log sampling for high-volume scenarios
type SamplingConfig struct {
	Thereafter int `json:"thereafter,omitempty"` // 1 in N after initial burst
}