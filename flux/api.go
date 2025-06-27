package flux

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"zbz/cereal"
	"zbz/depot"
	"zbz/zlog"
)

// FluxOptions configures flux behavior
type FluxOptions struct {
	ThrottleDuration       *time.Duration // Override default throttle duration (nil = use default 100ms)
	SkipInitialCallback    bool           // If true, don't call callbacks on initial load, only on changes
	SkipSecurityValidation bool           // If true, skip file type/size validation for trusted sources  
	MaxFileSize            *int64         // Override default file size limit (nil = use default)
}

// Get loads content once using document-aware parsing based on file extension
func Get[T any](provider depot.DepotProvider, uri string) (T, error) {
	// Load content from depot provider
	content, err := provider.Get(uri)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to load content for uri '%s': %w", uri, err)
	}

	// Always validate content for security (using default settings)
	if err := validateContent(content, uri, nil); err != nil {
		var zero T
		return zero, fmt.Errorf("content validation failed for uri '%s': %w", uri, err)
	}

	// Parse content using document-aware parsing
	parsed, err := parseByExtension[T](content, uri)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to parse content for uri '%s': %w", uri, err)
	}

	zlog.Debug("Loaded content from depot", 
		zlog.String("uri", uri),
		zlog.String("provider", provider.GetProvider()))
	
	return parsed, nil
}

// Sync creates a reactive watcher using document-aware parsing with simple variadic callbacks
func Sync[T any](provider depot.DepotProvider, uri string, callbacks ...func(old, new T, err error)) (FluxContract, error) {
	return SyncWithOptions(provider, uri, FluxOptions{}, callbacks...)
}

// SyncWithOptions creates a reactive watcher with configuration options
func SyncWithOptions[T any](provider depot.DepotProvider, uri string, options FluxOptions, callbacks ...func(old, new T, err error)) (FluxContract, error) {
	if len(callbacks) == 0 {
		return nil, fmt.Errorf("at least one callback is required")
	}

	// Load initial content
	content, err := provider.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial content for uri '%s': %w", uri, err)
	}

	// Validate content if security validation is enabled
	if !options.SkipSecurityValidation {
		if err := validateContent(content, uri, options.MaxFileSize); err != nil {
			return nil, fmt.Errorf("initial content validation failed for uri '%s': %w", uri, err)
		}
	}

	// Parse initial content using document-aware parsing
	parsed, err := parseByExtension[T](content, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initial content for uri '%s': %w", uri, err)
	}

	// Call all callbacks immediately with initial value (no error on successful load)
	var zero T
	executeCallbacks(callbacks, zero, parsed, nil)

	// Create simple watcher with options
	watcher := &simpleWatcher[T]{
		provider:  provider,
		uri:       uri,
		callbacks: callbacks,
		lastValue: &parsed,
		state:     stateActive,
		options:   options,
	}
	
	// Apply throttle duration from options
	if options.ThrottleDuration != nil {
		watcher.throttleDuration = *options.ThrottleDuration
	} else {
		watcher.throttleDuration = 100 * time.Millisecond
	}

	// Start watching
	if err := watcher.startWatching(); err != nil {
		return nil, fmt.Errorf("failed to start watching: %w", err)
	}

	zlog.Info("Created flux watcher", 
		zlog.String("uri", uri),
		zlog.String("provider", provider.GetProvider()),
		zlog.Int("callbacks", len(callbacks)))
	
	return watcher, nil
}

// SyncCollection watches all files matching a pattern with typed homogeneous collections
func SyncCollection[T any](provider depot.DepotProvider, pattern string, callbacks ...func(old, new map[string]T, err error)) (FluxContract, error) {
	return SyncCollectionWithOptions(provider, pattern, FluxOptions{}, callbacks...)
}

// SyncCollectionWithOptions watches all files matching a pattern with configuration options
func SyncCollectionWithOptions[T any](provider depot.DepotProvider, pattern string, options FluxOptions, callbacks ...func(old, new map[string]T, err error)) (FluxContract, error) {
	if len(callbacks) == 0 {
		return nil, fmt.Errorf("at least one callback is required")
	}

	// Create typed collection watcher with options
	watcher := &collectionWatcher[T]{
		provider:  provider,
		pattern:   pattern,
		callbacks: callbacks,
		currentFiles: make(map[string]T),
		subscriptions: make(map[string]depot.SubscriptionID),
		state:     stateActive,
		options:   options,
	}
	
	// Apply throttle duration from options
	if options.ThrottleDuration != nil {
		watcher.throttleDuration = *options.ThrottleDuration
	} else {
		watcher.throttleDuration = 100 * time.Millisecond
	}

	// Start watching
	if err := watcher.startWatching(); err != nil {
		return nil, fmt.Errorf("failed to start collection watching: %w", err)
	}

	zlog.Info("Created typed collection watcher", 
		zlog.String("pattern", pattern),
		zlog.String("provider", provider.GetProvider()),
		zlog.Int("callbacks", len(callbacks)))
	
	return watcher, nil
}

// executeCallbacks runs all callbacks independently with error information
func executeCallbacks[T any](callbacks []func(old, new T, err error), old, new T, err error) {
	for _, callback := range callbacks {
		// Each callback is independent - if one panics, others still run
		func() {
			defer func() {
				if r := recover(); r != nil {
					zlog.Warn("Callback panic recovered", 
						zlog.Any("panic", r))
				}
			}()
			callback(old, new, err)
		}()
	}
}

// parseByExtension uses document-aware parsing based on file extension
func parseByExtension[T any](content []byte, uri string) (T, error) {
	var result T
	ext := strings.ToLower(filepath.Ext(uri))
	
	// Special handling for primitive types
	switch any(result).(type) {
	case string:
		if ext == ".txt" || ext == ".md" || ext == "" {
			// For string types, return content as string
			return any(string(content)).(T), nil
		}
	case []byte:
		// For byte types, return raw content
		return any(content).(T), nil
	}
	
	// For struct types, use appropriate serializer based on extension
	switch ext {
	case ".json":
		if err := cereal.JSON.Unmarshal(content, &result); err != nil {
			return result, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := cereal.YAML.Unmarshal(content, &result); err != nil {
			return result, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".toml":
		if err := cereal.TOML.Unmarshal(content, &result); err != nil {
			return result, fmt.Errorf("failed to parse TOML: %w", err)
		}
	default:
		return result, fmt.Errorf("unsupported file extension '%s' for structured data", ext)
	}
	
	return result, nil
}

// validateContent performs security validation on content
func validateContent(content []byte, uri string, maxFileSize *int64) error {
	return validateContentWithSecurity(content, uri, maxFileSize)
}