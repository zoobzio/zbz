package flux

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CerealFluxOptions extends FluxOptions with cereal-specific configuration
type CerealFluxOptions struct {
	FluxOptions
	
	// Cereal-specific options
	EnableScoping     bool     // Enable field-level scoping for configuration
	DefaultFormat     string   // Default format when type detection fails
	AllowedFormats    []string // Restrict allowed formats for security
	ScopedPermissions []string // Default permissions for scoped operations
}

// parseCerealJSON parses JSON content using cereal
func parseCerealJSON[T any](content []byte) (any, error) {
	// Use cereal's JSON provider
	config := cereal.DefaultConfig()
	config.DefaultFormat = "json"
	contract := cereal.NewJSONProvider(config)
	
	var result T
	if err := contract.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("cereal JSON parse failed: %w", err)
	}
	return result, nil
}

// parseCerealYAML parses YAML content using cereal (would require YAML provider)
func parseCerealYAML[T any](content []byte) (any, error) {
	// This would use the cereal YAML provider
	// For now, placeholder implementation
	return nil, fmt.Errorf("cereal YAML parser not yet implemented")
}

// parseCerealRaw passes through raw bytes using cereal
func parseCerealRaw(content []byte) (any, error) {
	config := cereal.DefaultConfig()
	config.DefaultFormat = "raw"
	contract := cereal.NewRawProvider(config)
	
	var result []byte
	if err := contract.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("cereal raw parse failed: %w", err)
	}
	return result, nil
}

// parseCerealString converts to string using cereal
func parseCerealString(content []byte) (any, error) {
	config := cereal.DefaultConfig()
	config.DefaultFormat = "string"
	contract := cereal.NewStringProvider(config)
	
	var result string
	if err := contract.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("cereal string parse failed: %w", err)
	}
	return result, nil
}

// selectCerealParserForKey selects appropriate cereal parser based on file extension and type
func selectCerealParserForKey[T any](key string, options CerealFluxOptions) (func([]byte) (any, error), error) {
	ext := strings.ToLower(filepath.Ext(key))
	
	// Check if format is allowed (if restrictions are set)
	if len(options.AllowedFormats) > 0 {
		allowed := false
		for _, allowedFormat := range options.AllowedFormats {
			if ext == "."+allowedFormat {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("format %s not allowed by security policy", ext)
		}
	}
	
	// Select parser based on extension and type
	switch ext {
	case ".json":
		return parseCerealJSON[T], nil
	case ".yaml", ".yml":
		return parseCerealYAML[T], nil
	case ".txt", ".md":
		// For text files, determine if we want string or raw bytes
		var dummy T
		if fmt.Sprintf("%T", dummy) == "string" {
			return parseCerealString, nil
		}
		return parseCerealRaw, nil
	default:
		// Unknown extension - use default format or type-based detection
		var dummy T
		switch any(dummy).(type) {
		case []byte:
			return parseCerealRaw, nil
		case string:
			return parseCerealString, nil
		default:
			// Default to JSON for struct types
			if options.DefaultFormat == "yaml" {
				return parseCerealYAML[T], nil
			}
			return parseCerealJSON[T], nil
		}
	}
}

// CerealSync creates a reactive watcher using cereal for serialization
func CerealSync[T any](contract *hodor.HodorContract, key string, callback func(old, new T), options ...CerealFluxOptions) (FluxContract, error) {
	// Apply options (default if none provided)
	var opts CerealFluxOptions
	if len(options) > 0 {
		opts = options[0]
	}
	
	// Set cereal-specific defaults
	if opts.DefaultFormat == "" {
		opts.DefaultFormat = "json"
	}
	
	// Select appropriate cereal parser based on type T and key extension
	parseFunc, err := selectCerealParserForKey[T](key, opts)
	if err != nil {
		return nil, fmt.Errorf("no suitable cereal parser for key '%s': %w", key, err)
	}
	
	// Load initial content from hodor contract
	content, err := contract.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial content for key '%s': %w", key, err)
	}
	
	// Validate content if security validation is enabled
	if !opts.SkipSecurityValidation {
		if err := validateContent(content, key, opts.MaxFileSize); err != nil {
			return nil, fmt.Errorf("content validation failed for key '%s': %w", key, err)
		}
	}
	
	// Parse initial content using cereal
	parsed, err := parseFunc(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initial content with cereal for key '%s': %w", key, err)
	}
	
	typedParsed, ok := parsed.(T)
	if !ok {
		return nil, fmt.Errorf("parsed content type mismatch for key '%s': expected %T, got %T", key, *new(T), parsed)
	}
	
	// Create watcher contract with cereal-enhanced parsing
	watcher := &watcherContract[T]{
		key:         key,
		hodorClient: contract,
		parseFunc:   parseFunc,
		callback:    callback,
		options:     opts.FluxOptions,
		cerealOpts:  opts,
		current:     typedParsed,
	}
	
	// Register with hodor subscription system
	subscription, err := contract.Subscribe(key, watcher.handleChange)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to changes for key '%s': %w", key, err)
	}
	
	watcher.subscription = subscription
	
	// Trigger initial callback if not skipped
	if !opts.SkipInitialCallback {
		var zero T
		callback(zero, typedParsed)
	}
	
	return watcher, nil
}

// Enhanced watcher contract with cereal support
type watcherContract[T any] struct {
	key          string
	hodorClient  *hodor.HodorContract
	parseFunc    func([]byte) (any, error)
	callback     func(old, new T)
	options      FluxOptions
	cerealOpts   CerealFluxOptions
	subscription interface{} // hodor subscription handle
	current      T
}

// handleChange processes file changes using cereal parsing
func (w *watcherContract[T]) handleChange(newContent []byte) {
	// Parse new content using cereal
	parsed, err := w.parseFunc(newContent)
	if err != nil {
		// Log error but don't crash the watcher
		fmt.Printf("Cereal parse error for key '%s': %v\n", w.key, err)
		return
	}
	
	typedParsed, ok := parsed.(T)
	if !ok {
		fmt.Printf("Type assertion failed for key '%s': expected %T, got %T\n", w.key, *new(T), parsed)
		return
	}
	
	// Store old value and update current
	old := w.current
	w.current = typedParsed
	
	// Trigger callback
	w.callback(old, typedParsed)
}

// Stop implements FluxContract
func (w *watcherContract[T]) Stop() error {
	// Unsubscribe from hodor
	if w.subscription != nil {
		return w.hodorClient.Unsubscribe(w.key, w.subscription)
	}
	return nil
}

// Key implements FluxContract
func (w *watcherContract[T]) Key() string {
	return w.key
}

// IsActive implements FluxContract
func (w *watcherContract[T]) IsActive() bool {
	return w.subscription != nil
}

// GetCurrent returns the current parsed value
func (w *watcherContract[T]) GetCurrent() T {
	return w.current
}

// UpdateWithScoping updates the watched file with scoped serialization
func (w *watcherContract[T]) UpdateWithScoping(newValue T, userPermissions []string) error {
	if !w.cerealOpts.EnableScoping {
		return fmt.Errorf("scoped updates not enabled for this watcher")
	}
	
	// Serialize using cereal with scoping
	var serialized []byte
	var err error
	
	ext := strings.ToLower(filepath.Ext(w.key))
	switch ext {
	case ".json":
		// Use cereal JSON provider with scoping
		config := cereal.DefaultConfig()
		config.EnableScoping = true
		contract := cereal.NewJSONProvider(config)
		serialized, err = contract.MarshalScoped(newValue, userPermissions)
	case ".yaml", ".yml":
		// Use cereal YAML provider with scoping
		// This would require the YAML provider to be implemented
		return fmt.Errorf("scoped YAML serialization not yet implemented")
	default:
		return fmt.Errorf("scoped serialization not supported for file type: %s", ext)
	}
	
	if err != nil {
		return fmt.Errorf("failed to serialize with scoping: %w", err)
	}
	
	// Update in hodor
	return w.hodorClient.Set(w.key, serialized, 0) // No TTL for configuration files
}

// Example usage functions

// SetupCerealFlux configures flux to use cereal for all serialization
func SetupCerealFlux() {
	// This would replace the default flux adapters with cereal-based ones
	// Implementation would involve updating the global parser registry
}

// CreateConfigWatcher creates a configuration file watcher with cereal
func CreateConfigWatcher[T any](storageContract *hodor.HodorContract, configPath string, onUpdate func(old, new T)) (FluxContract, error) {
	options := CerealFluxOptions{
		FluxOptions: FluxOptions{
			SkipInitialCallback: false,
		},
		EnableScoping:  false, // Config files typically don't need scoping
		DefaultFormat:  "yaml", // Prefer YAML for config files
		AllowedFormats: []string{"yaml", "yml", "json"}, // Security restriction
	}
	
	return CerealSync(storageContract, configPath, onUpdate, options)
}