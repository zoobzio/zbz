package docula

import (
	"zbz/hodor"
)

// DoculaContract defines the configuration for the living documentation system
type DoculaContract struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	
	// Hodor storage for markdown content
	Storage *hodor.HodorContract `yaml:"storage,omitempty"`
	
	// UI configuration
	DocsUI *DocsUIConfig `yaml:"docs_ui,omitempty"`
	
	// Site configurations
	Sites []SiteConfig `yaml:"sites,omitempty"`
}

// DocsUIConfig configures the embedded documentation UI
type DocsUIConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`     // Default: "/docs"
	Engine  string `yaml:"engine"`   // "scalar", "swagger", "redoc"
	Title   string `yaml:"title,omitempty"`
}

// SiteConfig defines a documentation site template
type SiteConfig struct {
	Template    string            `yaml:"template"`    // "docs", "blog", "kb"
	BasePath    string            `yaml:"base_path"`   // URL prefix
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Features    map[string]bool   `yaml:"features"`    // Toggle features
	Theme       map[string]string `yaml:"theme"`       // Colors, fonts
}

// Docula creates a new documentation service from the contract
func (c DoculaContract) Docula() *Service {
	return NewService(c)
}