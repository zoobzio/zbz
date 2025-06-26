package flux

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"zbz/hodor"
	"zbz/zlog"
)

// CollectionWatcher manages watching multiple files matching a pattern
type CollectionWatcher struct {
	contract    *hodor.HodorContract
	pattern     string
	callback    func(old, new map[string][]byte)
	
	// Current state
	mu           sync.RWMutex
	currentFiles map[string][]byte
	subscriptions map[string]hodor.SubscriptionID
	
	// Configuration
	throttleDuration time.Duration
	lastUpdate      time.Time
	
	// Control
	active bool
}

// SyncCollection watches all files matching a pattern in hodor storage
func SyncCollection(contract *hodor.HodorContract, pattern string, callback func(old, new map[string][]byte), options ...FluxOptions) (*CollectionWatcher, error) {
	// Apply options
	var opts FluxOptions
	if len(options) > 0 {
		opts = options[0]
	}
	
	// Set default throttle duration
	throttleDuration := 100 * time.Millisecond
	if opts.ThrottleDuration != nil {
		throttleDuration = *opts.ThrottleDuration
	}
	
	watcher := &CollectionWatcher{
		contract:         contract,
		pattern:          pattern,
		callback:         callback,
		currentFiles:     make(map[string][]byte),
		subscriptions:    make(map[string]hodor.SubscriptionID),
		throttleDuration: throttleDuration,
		active:           true,
	}
	
	// Load initial files
	if err := watcher.loadInitialFiles(); err != nil {
		return nil, fmt.Errorf("failed to load initial files: %w", err)
	}
	
	// Set up subscriptions for current files
	if err := watcher.setupSubscriptions(); err != nil {
		return nil, fmt.Errorf("failed to setup subscriptions: %w", err)
	}
	
	// Call initial callback if not skipped
	if !opts.SkipInitialCallback {
		callback(make(map[string][]byte), watcher.currentFiles)
	}
	
	zlog.Info("Created collection watcher", 
		zlog.String("pattern", pattern),
		zlog.String("provider", contract.GetProvider()),
		zlog.Int("initial_files", len(watcher.currentFiles)))
	
	return watcher, nil
}

// loadInitialFiles discovers and loads all files matching the pattern
func (cw *CollectionWatcher) loadInitialFiles() error {
	// List all files in storage
	allFiles, err := cw.contract.List("")
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}
	
	// Filter files matching pattern
	matchingFiles := cw.filterFiles(allFiles)
	
	// Load content for each matching file
	for _, filePath := range matchingFiles {
		content, err := cw.contract.Get(filePath)
		if err != nil {
			zlog.Warn("Failed to load file in collection", 
				zlog.String("file", filePath), 
				zlog.Err(err))
			continue
		}
		
		cw.currentFiles[filePath] = content
		zlog.Debug("Loaded file into collection", zlog.String("file", filePath))
	}
	
	return nil
}

// filterFiles returns files that match the pattern
func (cw *CollectionWatcher) filterFiles(files []string) []string {
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

// setupSubscriptions creates hodor subscriptions for all current files
func (cw *CollectionWatcher) setupSubscriptions() error {
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
func (cw *CollectionWatcher) subscribeToFile(filePath string) error {
	subscriptionID, err := cw.contract.Subscribe(filePath, func(event hodor.ChangeEvent) {
		cw.handleFileChange(filePath, event)
	})
	if err != nil {
		return err
	}
	
	cw.subscriptions[filePath] = subscriptionID
	return nil
}

// handleFileChange processes individual file changes and triggers collection updates
func (cw *CollectionWatcher) handleFileChange(filePath string, event hodor.ChangeEvent) {
	if !cw.active {
		return
	}
	
	// Throttle updates to avoid excessive callback calls
	cw.mu.Lock()
	now := time.Now()
	if now.Sub(cw.lastUpdate) < cw.throttleDuration {
		cw.mu.Unlock()
		return
	}
	cw.lastUpdate = now
	cw.mu.Unlock()
	
	// Small delay to allow multiple rapid changes to accumulate
	time.AfterFunc(50*time.Millisecond, func() {
		cw.processChanges()
	})
}

// processChanges scans for all changes and triggers the callback
func (cw *CollectionWatcher) processChanges() {
	if !cw.active {
		return
	}
	
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	// Capture old state
	oldFiles := make(map[string][]byte)
	for k, v := range cw.currentFiles {
		oldFiles[k] = append([]byte(nil), v...) // Deep copy
	}
	
	// Scan for current files matching pattern
	allFiles, err := cw.contract.List("")
	if err != nil {
		zlog.Warn("Failed to list files during change processing", zlog.Err(err))
		return
	}
	
	matchingFiles := cw.filterFiles(allFiles)
	newFiles := make(map[string][]byte)
	
	// Load current content for all matching files
	for _, filePath := range matchingFiles {
		content, err := cw.contract.Get(filePath)
		if err != nil {
			zlog.Warn("Failed to load file during change processing", 
				zlog.String("file", filePath), 
				zlog.Err(err))
			continue
		}
		newFiles[filePath] = content
		
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
			cw.contract.Unsubscribe(subscriptionID)
			delete(cw.subscriptions, filePath)
		}
	}
	
	// Update current state
	cw.currentFiles = newFiles
	
	// Check if there are actual changes
	if cw.hasChanges(oldFiles, newFiles) {
		zlog.Debug("Collection changed, triggering callback", 
			zlog.Int("old_count", len(oldFiles)),
			zlog.Int("new_count", len(newFiles)))
		
		// Trigger callback
		go cw.callback(oldFiles, newFiles)
	}
}

// hasChanges compares old and new file collections
func (cw *CollectionWatcher) hasChanges(old, new map[string][]byte) bool {
	// Different number of files
	if len(old) != len(new) {
		return true
	}
	
	// Check for content changes
	for path, newContent := range new {
		oldContent, exists := old[path]
		if !exists {
			return true // New file
		}
		
		if string(oldContent) != string(newContent) {
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

// Stop stops the collection watcher and cleans up subscriptions
func (cw *CollectionWatcher) Stop() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	cw.active = false
	
	// Unsubscribe from all files
	for filePath, subscriptionID := range cw.subscriptions {
		if err := cw.contract.Unsubscribe(subscriptionID); err != nil {
			zlog.Warn("Failed to unsubscribe from file", 
				zlog.String("file", filePath), 
				zlog.Err(err))
		}
	}
	
	cw.subscriptions = make(map[string]hodor.SubscriptionID)
	cw.currentFiles = make(map[string][]byte)
	
	zlog.Info("Stopped collection watcher", zlog.String("pattern", cw.pattern))
	return nil
}

// GetCurrentFiles returns a copy of the current file collection
func (cw *CollectionWatcher) GetCurrentFiles() map[string][]byte {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	
	result := make(map[string][]byte)
	for k, v := range cw.currentFiles {
		result[k] = append([]byte(nil), v...) // Deep copy
	}
	return result
}

// AddFile manually adds a file to the collection (useful for testing)
func (cw *CollectionWatcher) AddFile(path string, content []byte) error {
	// Store in hodor (this will trigger the normal subscription flow)
	return cw.contract.Set(path, content, 0)
}

// UpdateFile manually updates a file in the collection (useful for testing)
func (cw *CollectionWatcher) UpdateFile(path string, content []byte) error {
	// Store in hodor (this will trigger the normal subscription flow)
	return cw.contract.Set(path, content, 0)
}

// RemoveFile manually removes a file from the collection (useful for testing)
func (cw *CollectionWatcher) RemoveFile(path string) error {
	// Delete from hodor (this will trigger the normal subscription flow)
	return cw.contract.Delete(path)
}