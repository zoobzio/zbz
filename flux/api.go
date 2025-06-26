package flux

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"zbz/hodor"
	"zbz/zlog"
)

// FluxOptions configures contract behavior for hodor-based watching
type FluxOptions struct {
	ThrottleDuration       *time.Duration // Override default throttle duration (nil = use default 100ms)
	SkipInitialCallback    bool           // If true, don't call handlers on initial load, only on changes
	SkipSecurityValidation bool           // If true, skip file type/size validation for trusted sources  
	MaxFileSize            *int64         // Override default file size limit (nil = use default)
}

// Sync creates a reactive watcher for a key in a hodor contract
func Sync[T any](contract *hodor.HodorContract, key string, callback func(old, new T), options ...FluxOptions) (FluxContract, error) {
	// Apply options (default if none provided)
	var opts FluxOptions
	if len(options) > 0 {
		opts = options[0]
	}

	// Select appropriate parser based on type T and key extension
	parseFunc, err := selectParserForKey[T](key)
	if err != nil {
		return nil, fmt.Errorf("no suitable parser for key '%s': %w", key, err)
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

	// Parse initial content
	parsed, err := parseFunc(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initial content for key '%s': %w", key, err)
	}

	typedValue, ok := parsed.(T)
	if !ok {
		return nil, fmt.Errorf("parsed value type mismatch for key '%s'", key)
	}

	// Call callback immediately with initial value (unless skipped)
	if !opts.SkipInitialCallback {
		var zero T
		callback(zero, typedValue)
	}

	// Create hodor-based watcher
	watcher := &watcher[T]{
		contract:               contract,
		key:                    key,
		parseFunc:              parseFunc,
		callback:               callback,
		lastValue:              &typedValue,
		state:                  stateActive,
		throttleDuration:       opts.ThrottleDuration,
		skipSecurityValidation: opts.SkipSecurityValidation,
		maxFileSize:            opts.MaxFileSize,
	}

	// Start cloud watching using hodor's Subscribe
	if err := watcher.startWatching(); err != nil {
		return nil, fmt.Errorf("failed to start hodor watching for key '%s': %w", key, err)
	}

	zlog.Info("Created hodor watcher", 
		zlog.String("key", key),
		zlog.String("provider", contract.GetProvider()))
	
	return watcher, nil
}

// Get loads content once from a hodor contract without watching for changes
func Get[T any](contract *hodor.HodorContract, key string, options ...FluxOptions) (T, error) {
	// Apply options (default if none provided)
	var opts FluxOptions
	if len(options) > 0 {
		opts = options[0]
	}

	// Select appropriate parser based on type T and key extension
	parseFunc, err := selectParserForKey[T](key)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("no suitable parser for key '%s': %w", key, err)
	}

	// Load content from hodor contract
	content, err := contract.Get(key)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to load content for key '%s': %w", key, err)
	}

	// Validate content if security validation is enabled
	if !opts.SkipSecurityValidation {
		if err := validateContent(content, key, opts.MaxFileSize); err != nil {
			var zero T
			return zero, fmt.Errorf("content validation failed for key '%s': %w", key, err)
		}
	}

	// Parse content
	parsed, err := parseFunc(content)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to parse content for key '%s': %w", key, err)
	}

	typedValue, ok := parsed.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("parsed value type mismatch for key '%s'", key)
	}

	zlog.Debug("Loaded content from hodor", 
		zlog.String("key", key),
		zlog.String("provider", contract.GetProvider()))
	
	return typedValue, nil
}

// SyncWithHandlers creates a reactive watcher with a handler pipeline
func SyncWithHandlers[T any](contract *hodor.HodorContract, key string, handlers []HandlerFunc[T], options ...FluxOptions) (FluxContract, error) {
	// Convert handler pipeline to simple callback
	callback := func(old, new T) {
		event := NewFluxEvent(key, old, new, "update", contract.GetProvider())
		event.Size = int64(len(fmt.Sprintf("%+v", new))) // Approximate size
		
		pipeline := NewPipeline(handlers...)
		pipeline.Execute(event)
		
		// Log any errors from handlers
		if event.HasErrors() {
			for _, err := range event.Errors() {
				zlog.Warn("Handler error", 
					zlog.String("key", key),
					zlog.Err(err))
			}
		}
	}
	
	return Sync(contract, key, callback, options...)
}


// selectParserForKey chooses the appropriate parser based on type T and key extension
func selectParserForKey[T any](key string) (func([]byte) (any, error), error) {
	ext := strings.ToLower(filepath.Ext(key))

	// Check what type T is and match with appropriate parser
	switch any(*new(T)).(type) {
	case []byte:
		return parseBytes, nil
	case string:
		if ext == ".md" || ext == ".txt" {
			return parseText, nil
		}
		return nil, fmt.Errorf("string type not supported for %s files", ext)
	default:
		// Struct types - select parser by extension
		switch ext {
		case ".yaml", ".yml":
			return parseYAML[T], nil
		case ".json":
			return parseJSON[T], nil
		default:
			return nil, fmt.Errorf("no parser available for %s files with struct type", ext)
		}
	}
}

// validateContent performs comprehensive security validation on content
func validateContent(content []byte, key string, maxFileSize *int64) error {
	// Validate using our comprehensive security functions
	return validateContentWithSecurity(content, key, maxFileSize)
}