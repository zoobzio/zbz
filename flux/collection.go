package flux

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"zbz/depot"
	"zbz/zlog"
)

// collectionWatcher manages watching multiple files matching a pattern with typed parsing
type collectionWatcher[T any] struct {
	provider  depot.DepotProvider
	pattern   string
	callbacks []func(old, new map[string]T, err error)
	options   FluxOptions
	
	// State management
	mu            sync.RWMutex
	currentFiles  map[string]T
	subscriptions map[string]depot.SubscriptionID
	state         watcherState
	
	// Throttling
	throttleTimer    *time.Timer
	throttleDuration time.Duration
}

// startWatching initializes the collection watcher
func (cw *collectionWatcher[T]) startWatching() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	// Load initial files and validate homogeneity
	if err := cw.loadInitialFiles(); err != nil {
		return fmt.Errorf("failed to load initial files: %w", err)
	}
	
	// Set up subscriptions for current files
	if err := cw.setupSubscriptions(); err != nil {
		return fmt.Errorf("failed to setup subscriptions: %w", err)
	}
	
	// Call initial callbacks with empty old state (no error on successful load)
	if !cw.options.SkipInitialCallback {
		emptyState := make(map[string]T)
		executeCollectionCallbacks(cw.callbacks, emptyState, cw.currentFiles, nil)
	}
	
	return nil
}

// loadInitialFiles discovers and loads all files matching the pattern
func (cw *collectionWatcher[T]) loadInitialFiles() error {
	// List all files in storage
	allFiles, err := cw.provider.List("")
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}
	
	// Filter files matching pattern
	matchingFiles := cw.filterFiles(allFiles)
	
	// Validate homogeneous file extensions
	if err := cw.validateHomogeneousCollection(matchingFiles); err != nil {
		return err
	}
	
	// Load and parse content for each matching file
	for _, filePath := range matchingFiles {
		content, err := cw.provider.Get(filePath)
		if err != nil {
			zlog.Warn("Failed to load file in collection", 
				zlog.String("file", filePath), 
				zlog.Err(err))
			continue
		}
		
		// Validate content if security validation is enabled
		if !cw.options.SkipSecurityValidation {
			if err := validateContent(content, filePath, cw.options.MaxFileSize); err != nil {
				zlog.Warn("Security validation failed for file in collection", 
					zlog.String("file", filePath), 
					zlog.Err(err))
				continue
			}
		}

		// Parse using document-aware parsing
		parsed, err := parseByExtension[T](content, filePath)
		if err != nil {
			zlog.Warn("Failed to parse file in collection", 
				zlog.String("file", filePath), 
				zlog.Err(err))
			continue
		}
		
		cw.currentFiles[filePath] = parsed
		zlog.Debug("Loaded file into typed collection", zlog.String("file", filePath))
	}
	
	return nil
}

// validateHomogeneousCollection ensures all files have the same extension
func (cw *collectionWatcher[T]) validateHomogeneousCollection(files []string) error {
	if len(files) == 0 {
		return nil // Empty collections are valid
	}
	
	// Get extension from first file
	firstExt := strings.ToLower(filepath.Ext(files[0]))
	
	// Validate all files have the same extension
	for _, file := range files[1:] {
		ext := strings.ToLower(filepath.Ext(file))
		if ext != firstExt {
			return fmt.Errorf("heterogeneous collection detected: mixed extensions '%s' and '%s' - collections must contain files of the same type", firstExt, ext)
		}
	}
	
	// Validate extension is supported for structured data
	switch firstExt {
	case ".json", ".yaml", ".yml", ".toml":
		// Supported structured formats
	case ".txt", ".md", "":
		// Supported text formats
	default:
		return fmt.Errorf("unsupported file extension '%s' for collections", firstExt)
	}
	
	zlog.Debug("Validated homogeneous collection", 
		zlog.String("extension", firstExt),
		zlog.Int("file_count", len(files)))
	
	return nil
}

// filterFiles returns files that match the pattern
func (cw *collectionWatcher[T]) filterFiles(files []string) []string {
	var matching []string
	
	for _, file := range files {
		// Handle glob patterns
		if strings.Contains(cw.pattern, "*") {
			if matched, _ := filepath.Match(cw.pattern, file); matched {
				matching = append(matching, file)
			}
		} else if strings.Contains(cw.pattern, ".") {
			// Handle extension patterns like ".md"
			if strings.HasSuffix(file, cw.pattern) {
				matching = append(matching, file)
			}
		} else {
			// Handle prefix patterns
			if strings.HasPrefix(file, cw.pattern) {
				matching = append(matching, file)
			}
		}
	}
	
	return matching
}

// setupSubscriptions creates depot subscriptions for all current files
func (cw *collectionWatcher[T]) setupSubscriptions() error {
	for filePath := range cw.currentFiles {
		if err := cw.subscribeToFile(filePath); err != nil {
			zlog.Warn("Failed to subscribe to file", 
				zlog.String("file", filePath), 
				zlog.Err(err))
		}
	}
	return nil
}

// subscribeToFile creates a subscription for a specific file
func (cw *collectionWatcher[T]) subscribeToFile(filePath string) error {
	subscriptionID, err := cw.provider.Subscribe(filePath, func(event depot.ChangeEvent) {
		cw.handleDepotEvent(filePath, event)
	})
	if err != nil {
		return err
	}
	
	cw.subscriptions[filePath] = subscriptionID
	return nil
}

// handleDepotEvent processes depot change events for collection files
func (cw *collectionWatcher[T]) handleDepotEvent(filePath string, event depot.ChangeEvent) {
	// All events (create, update, delete) are relevant for collections
	// We'll let processCollectionChanges handle the full scan
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	// Skip if dismissed or paused
	if cw.state == stateDismissed || cw.state == statePaused {
		return
	}
	
	// Throttle rapid changes
	if cw.throttleTimer != nil {
		cw.throttleTimer.Stop()
	}
	
	cw.throttleTimer = time.AfterFunc(cw.throttleDuration, func() {
		cw.processCollectionChanges()
	})
}

// handleFileChange processes individual file changes (legacy method for internal use)
func (cw *collectionWatcher[T]) handleFileChange(filePath string, content []byte) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	// Skip if dismissed or paused
	if cw.state == stateDismissed || cw.state == statePaused {
		return
	}
	
	// Throttle rapid changes
	if cw.throttleTimer != nil {
		cw.throttleTimer.Stop()
	}
	
	cw.throttleTimer = time.AfterFunc(cw.throttleDuration, func() {
		cw.processCollectionChanges()
	})
}

// processCollectionChanges scans for all changes and triggers callbacks
func (cw *collectionWatcher[T]) processCollectionChanges() {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	// Skip if dismissed or paused
	if cw.state == stateDismissed || cw.state == statePaused {
		return
	}
	
	// Capture old state
	oldFiles := make(map[string]T)
	for k, v := range cw.currentFiles {
		oldFiles[k] = v
	}
	
	// Scan for current files matching pattern
	allFiles, err := cw.provider.List("")
	if err != nil {
		zlog.Warn("Failed to list files during change processing", zlog.Err(err))
		return
	}
	
	matchingFiles := cw.filterFiles(allFiles)
	
	// Validate homogeneity of new file set
	if err := cw.validateHomogeneousCollection(matchingFiles); err != nil {
		zlog.Error("Collection homogeneity violation during update", zlog.Err(err))
		// Enter recovery mode and notify callbacks of error
		cw.state = stateRecovering
		executeCollectionCallbacks(cw.callbacks, oldFiles, oldFiles, err)
		return
	}
	
	newFiles := make(map[string]T)
	
	// Load and parse current content for all matching files
	for _, filePath := range matchingFiles {
		content, err := cw.provider.Get(filePath)
		if err != nil {
			zlog.Warn("Failed to load file during change processing", 
				zlog.String("file", filePath), 
				zlog.Err(err))
			continue
		}
		
		// Validate content if security validation is enabled
		if !cw.options.SkipSecurityValidation {
			if err := validateContent(content, filePath, cw.options.MaxFileSize); err != nil {
				zlog.Warn("Security validation failed during change processing", 
					zlog.String("file", filePath), 
					zlog.Err(err))
				// Enter recovery mode on security error and notify callbacks
				cw.state = stateRecovering
				executeCollectionCallbacks(cw.callbacks, oldFiles, oldFiles, fmt.Errorf("security validation failed in file %s: %w", filePath, err))
				return
			}
		}

		// Parse using document-aware parsing
		parsed, err := parseByExtension[T](content, filePath)
		if err != nil {
			zlog.Warn("Failed to parse file during change processing", 
				zlog.String("file", filePath), 
				zlog.Err(err))
			// Enter recovery mode on parse error and notify callbacks
			cw.state = stateRecovering
			executeCollectionCallbacks(cw.callbacks, oldFiles, oldFiles, fmt.Errorf("parse error in file %s: %w", filePath, err))
			return
		}
		
		newFiles[filePath] = parsed
		
		// Subscribe to new files
		if _, exists := cw.subscriptions[filePath]; !exists {
			if err := cw.subscribeToFile(filePath); err != nil {
				zlog.Warn("Failed to subscribe to new file", 
					zlog.String("file", filePath), 
					zlog.Err(err))
			}
		}
	}
	
	// Unsubscribe from removed files
	for filePath, subscriptionID := range cw.subscriptions {
		if _, exists := newFiles[filePath]; !exists {
			cw.provider.Unsubscribe(subscriptionID)
			delete(cw.subscriptions, filePath)
		}
	}
	
	// Exit recovery mode if we were in it
	if cw.state == stateRecovering {
		cw.state = stateActive
		zlog.Info("Collection recovered from error", zlog.String("pattern", cw.pattern))
	}
	
	// Update current state
	cw.currentFiles = newFiles
	
	// Check if there are actual changes
	if cw.hasChanges(oldFiles, newFiles) {
		zlog.Debug("Typed collection changed, triggering callbacks", 
			zlog.Int("old_count", len(oldFiles)),
			zlog.Int("new_count", len(newFiles)))
		
		// Execute all callbacks independently with no error (successful parse)
		executeCollectionCallbacks(cw.callbacks, oldFiles, newFiles, nil)
	}
}

// hasChanges compares old and new file collections
func (cw *collectionWatcher[T]) hasChanges(old, new map[string]T) bool {
	// Different number of files
	if len(old) != len(new) {
		return true
	}
	
	// Check for content changes (using string comparison for simplicity)
	for path, newValue := range new {
		oldValue, exists := old[path]
		if !exists {
			return true // New file
		}
		
		// Use string representation for comparison
		if fmt.Sprintf("%+v", oldValue) != fmt.Sprintf("%+v", newValue) {
			return true // Content changed
		}
	}
	
	// Check for removed files
	for path := range old {
		if _, exists := new[path]; !exists {
			return true // File removed
		}
	}
	
	return false
}

// FluxContract implementation for typed collections

// Resolve gets the current collection state
func (cw *collectionWatcher[T]) Resolve() (any, error) {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	
	// Return a copy of current files
	result := make(map[string]T)
	for k, v := range cw.currentFiles {
		result[k] = v
	}
	
	return result, nil
}

// Dismiss stops watching and cleans up
func (cw *collectionWatcher[T]) Dismiss() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	if cw.state == stateDismissed {
		return nil // Already dismissed
	}
	
	// Stop throttle timer
	if cw.throttleTimer != nil {
		cw.throttleTimer.Stop()
	}
	
	// Unsubscribe from all files
	for filePath, subscriptionID := range cw.subscriptions {
		if err := cw.provider.Unsubscribe(subscriptionID); err != nil {
			zlog.Warn("Failed to unsubscribe from file", 
				zlog.String("file", filePath), 
				zlog.Err(err))
		}
	}
	
	cw.subscriptions = make(map[string]depot.SubscriptionID)
	cw.currentFiles = make(map[string]T)
	cw.state = stateDismissed
	
	zlog.Info("Dismissed typed collection watcher", zlog.String("pattern", cw.pattern))
	return nil
}

// IsActive returns true if watching
func (cw *collectionWatcher[T]) IsActive() bool {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.state != stateDismissed
}

// IsRecovering returns true if in recovery mode
func (cw *collectionWatcher[T]) IsRecovering() bool {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.state == stateRecovering
}

// Pause stops callback execution
func (cw *collectionWatcher[T]) Pause() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	if cw.state == stateDismissed {
		return fmt.Errorf("cannot pause dismissed watcher")
	}
	
	cw.state = statePaused
	return nil
}

// Resume restarts callback execution
func (cw *collectionWatcher[T]) Resume() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	if cw.state == stateDismissed {
		return fmt.Errorf("cannot resume dismissed watcher")
	}
	
	cw.state = stateActive
	return nil
}

// IsPaused returns true if paused
func (cw *collectionWatcher[T]) IsPaused() bool {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.state == statePaused
}

// executeCollectionCallbacks runs all collection callbacks independently with error information
func executeCollectionCallbacks[T any](callbacks []func(old, new map[string]T, err error), old, new map[string]T, err error) {
	for _, callback := range callbacks {
		// Each callback is independent - if one panics, others still run
		func() {
			defer func() {
				if r := recover(); r != nil {
					zlog.Warn("Collection callback panic recovered", 
						zlog.Any("panic", r))
				}
			}()
			callback(old, new, err)
		}()
	}
}