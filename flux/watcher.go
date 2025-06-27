package flux

import (
	"fmt"
	"sync"
	"time"

	"zbz/depot"
	"zbz/zlog"
)

// watcherState represents the current state of a file watcher
type watcherState int

const (
	stateActive    watcherState = iota // Normal operation, processing callbacks
	stateRecovering                    // Parse error, waiting for valid file
	statePaused                        // Paused, watching but not calling callbacks
	stateDismissed                     // Shut down, no longer watching
)

// FluxContract provides control over a file watcher
type FluxContract interface {
	// Resolve gets the current value without triggering callback
	Resolve() (any, error)

	// Dismiss stops watching the file and cleans up resources
	Dismiss() error

	// IsActive returns true if watching (active, recovering, or paused)
	IsActive() bool

	// IsRecovering returns true if in recovery mode due to parse error
	IsRecovering() bool

	// Pause temporarily stops callback execution while continuing to watch
	Pause() error

	// Resume restarts callback execution from paused state
	Resume() error

	// IsPaused returns true if watcher is in paused state
	IsPaused() bool
}

// simpleWatcher implements FluxContract for depot-based watching with simple callbacks
type simpleWatcher[T any] struct {
	provider  depot.DepotProvider
	uri       string
	callbacks []func(old, new T, err error)
	options   FluxOptions
	
	// State management
	mu         sync.RWMutex
	lastValue  *T
	state      watcherState
	subscription depot.SubscriptionID
	
	// Throttling
	throttleTimer *time.Timer
	throttleDuration time.Duration
}

// startWatching begins watching the file through depot
func (w *simpleWatcher[T]) startWatching() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// Throttle duration is set during watcher creation from options
	
	// Subscribe to changes through depot
	subscription, err := w.provider.Subscribe(w.uri, func(event depot.ChangeEvent) {
		w.handleDepotEvent(event)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to uri '%s': %w", w.uri, err)
	}
	
	w.subscription = subscription
	return nil
}

// handleDepotEvent processes depot change events
func (w *simpleWatcher[T]) handleDepotEvent(event depot.ChangeEvent) {
	// Only process create/update events, ignore deletes for single file watching
	if event.Operation == "delete" {
		zlog.Debug("Ignoring delete event for single file watcher", 
			zlog.String("uri", w.uri))
		return
	}
	
	// Load current content from provider
	content, err := w.provider.Get(w.uri)
	if err != nil {
		zlog.Warn("Failed to load content after depot event", 
			zlog.String("uri", w.uri),
			zlog.Err(err))
		return
	}
	
	w.handleContentChange(content)
}

// handleContentChange processes file content changes
func (w *simpleWatcher[T]) handleContentChange(content []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// Skip if dismissed or paused
	if w.state == stateDismissed || w.state == statePaused {
		return
	}
	
	// Throttle rapid changes
	if w.throttleTimer != nil {
		w.throttleTimer.Stop()
	}
	
	w.throttleTimer = time.AfterFunc(w.throttleDuration, func() {
		w.processContentChange(content)
	})
}

// processContentChange parses and executes callbacks
func (w *simpleWatcher[T]) processContentChange(content []byte) {
	w.mu.Lock()
	oldValue := *w.lastValue
	w.mu.Unlock()

	// Validate content if security validation is enabled
	if !w.options.SkipSecurityValidation {
		if err := validateContent(content, w.uri, w.options.MaxFileSize); err != nil {
			// Enter recovery mode on security error
			w.mu.Lock()
			w.state = stateRecovering
			w.mu.Unlock()
			
			zlog.Warn("Security validation failed, entering recovery mode", 
				zlog.String("uri", w.uri),
				zlog.Err(err))
			
			// Notify callbacks of security error with current values
			executeCallbacks(w.callbacks, oldValue, oldValue, err)
			return
		}
	}

	// Parse new content using document-aware parsing
	newValue, err := parseByExtension[T](content, w.uri)
	if err != nil {
		// Enter recovery mode on parse error
		w.mu.Lock()
		w.state = stateRecovering
		w.mu.Unlock()
		
		zlog.Warn("Parse error, entering recovery mode", 
			zlog.String("uri", w.uri),
			zlog.Err(err))
		
		// Notify callbacks of parse error with current values
		executeCallbacks(w.callbacks, oldValue, oldValue, err)
		return
	}
	
	w.mu.Lock()
	oldValue := *w.lastValue
	*w.lastValue = newValue
	
	// Exit recovery mode if we were in it
	if w.state == stateRecovering {
		w.state = stateActive
		zlog.Info("Recovered from parse error", zlog.String("uri", w.uri))
	}
	w.mu.Unlock()
	
	// Execute all callbacks independently with no error (successful parse)
	executeCallbacks(w.callbacks, oldValue, newValue, nil)
}

// Resolve gets the current value
func (w *simpleWatcher[T]) Resolve() (any, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	if w.lastValue == nil {
		return nil, fmt.Errorf("no value available")
	}
	
	return *w.lastValue, nil
}

// Dismiss stops watching and cleans up
func (w *simpleWatcher[T]) Dismiss() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.state == stateDismissed {
		return nil // Already dismissed
	}
	
	// Stop throttle timer
	if w.throttleTimer != nil {
		w.throttleTimer.Stop()
	}
	
	// Unsubscribe from depot
	if err := w.provider.Unsubscribe(w.subscription); err != nil {
		zlog.Warn("Failed to unsubscribe", 
			zlog.String("uri", w.uri),
			zlog.Err(err))
	}
	
	w.state = stateDismissed
	zlog.Debug("Dismissed flux watcher", zlog.String("uri", w.uri))
	
	return nil
}

// IsActive returns true if watching
func (w *simpleWatcher[T]) IsActive() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state != stateDismissed
}

// IsRecovering returns true if in recovery mode
func (w *simpleWatcher[T]) IsRecovering() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state == stateRecovering
}

// Pause stops callback execution
func (w *simpleWatcher[T]) Pause() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.state == stateDismissed {
		return fmt.Errorf("cannot pause dismissed watcher")
	}
	
	w.state = statePaused
	return nil
}

// Resume restarts callback execution
func (w *simpleWatcher[T]) Resume() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.state == stateDismissed {
		return fmt.Errorf("cannot resume dismissed watcher")
	}
	
	w.state = stateActive
	return nil
}

// IsPaused returns true if paused
func (w *simpleWatcher[T]) IsPaused() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state == statePaused
}