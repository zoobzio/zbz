package flux

import (
	"fmt"
	"sync"
	"time"

	"zbz/hodor"
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

// watcher implements FluxContract for hodor-based cloud storage watching
type watcher[T any] struct {
	contract               *hodor.HodorContract
	key                    string
	parseFunc              func([]byte) (any, error)
	callback               func(old, new T)
	lastValue              *T
	state                  watcherState
	throttleTimer          *time.Timer
	throttleDuration       *time.Duration
	skipSecurityValidation bool
	maxFileSize            *int64
	subscriptionID         hodor.SubscriptionID
	mu                     sync.RWMutex
}

// Ensure watcher implements FluxContract
var _ FluxContract = (*watcher[any])(nil)

// startWatching begins cloud event watching through hodor
func (w *watcher[T]) startWatching() error {
	// Subscribe to hodor contract events for this key
	subID, err := w.contract.Subscribe(w.key, func(event hodor.ChangeEvent) {
		w.handleCloudEvent(event)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to hodor events: %w", err)
	}

	w.subscriptionID = subID
	zlog.Debug("Started hodor watching", 
		zlog.String("key", w.key),
		zlog.String("subscription_id", string(subID)))

	return nil
}

// handleCloudEvent processes events from hodor cloud storage
func (w *watcher[T]) handleCloudEvent(event hodor.ChangeEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if watcher is active
	if w.state == stateDismissed {
		return
	}

	// Apply throttling if configured
	if w.shouldThrottle() {
		w.scheduleThrottledUpdate(event)
		return
	}

	w.processCloudEvent(event)
}

// processCloudEvent handles the actual event processing
func (w *watcher[T]) processCloudEvent(event hodor.ChangeEvent) {
	// Skip processing if paused
	if w.state == statePaused {
		zlog.Debug("Skipping event - watcher paused", zlog.String("key", w.key))
		return
	}

	// Load new content from hodor
	content, err := w.contract.Get(w.key)
	if err != nil {
		w.handleError(fmt.Errorf("failed to load content after change event: %w", err))
		return
	}

	// Validate content if security validation is enabled
	if !w.skipSecurityValidation {
		if err := validateContent(content, w.key, w.maxFileSize); err != nil {
			w.handleError(fmt.Errorf("content validation failed: %w", err))
			return
		}
	}

	// Parse new content
	parsed, err := w.parseFunc(content)
	if err != nil {
		w.handleError(fmt.Errorf("failed to parse content: %w", err))
		return
	}

	newValue, ok := parsed.(T)
	if !ok {
		w.handleError(fmt.Errorf("parsed value type mismatch"))
		return
	}

	// Recovery from error state if we were in recovery
	if w.state == stateRecovering {
		w.state = stateActive
		zlog.Info("Recovered from error state", zlog.String("key", w.key))
	}

	// Call callback with old and new values
	var oldValue T
	if w.lastValue != nil {
		oldValue = *w.lastValue
	}

	w.callback(oldValue, newValue)
	w.lastValue = &newValue

	zlog.Debug("Processed cloud event", 
		zlog.String("key", w.key),
		zlog.String("operation", event.Operation))
}

// shouldThrottle checks if throttling should be applied
func (w *watcher[T]) shouldThrottle() bool {
	// Use per-watcher throttle if set, otherwise use service default
	duration := w.getThrottleDuration()
	if duration <= 0 {
		return false
	}

	// If timer is already running, we should throttle
	if w.throttleTimer != nil {
		return true
	}

	return false
}

// scheduleThrottledUpdate schedules an update after throttle duration
func (w *watcher[T]) scheduleThrottledUpdate(event hodor.ChangeEvent) {
	duration := w.getThrottleDuration()
	
	// Reset existing timer if any
	if w.throttleTimer != nil {
		w.throttleTimer.Stop()
	}

	// Schedule delayed processing
	w.throttleTimer = time.AfterFunc(duration, func() {
		w.mu.Lock()
		w.throttleTimer = nil
		w.mu.Unlock()
		
		w.processCloudEvent(event)
	})

	zlog.Debug("Throttled event", 
		zlog.String("key", w.key),
		zlog.Duration("delay", duration))
}

// getThrottleDuration returns the throttle duration for this watcher
func (w *watcher[T]) getThrottleDuration() time.Duration {
	if w.throttleDuration != nil {
		return *w.throttleDuration
	}
	// Use service default (100ms)
	return 100 * time.Millisecond
}

// handleError puts watcher into recovery state
func (w *watcher[T]) handleError(err error) {
	w.state = stateRecovering
	zlog.Warn("Watcher error - entering recovery state", 
		zlog.String("key", w.key),
		zlog.Err(err))
}

// FluxContract implementation

// Resolve gets the current value without triggering callback
func (w *watcher[T]) Resolve() (any, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.state == stateDismissed {
		return nil, fmt.Errorf("watcher is dismissed")
	}

	content, err := w.contract.Get(w.key)
	if err != nil {
		return nil, fmt.Errorf("failed to load content: %w", err)
	}

	return w.parseFunc(content)
}

// Dismiss stops watching and cleans up resources
func (w *watcher[T]) Dismiss() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.state == stateDismissed {
		return nil
	}

	// Stop throttle timer if running
	if w.throttleTimer != nil {
		w.throttleTimer.Stop()
		w.throttleTimer = nil
	}

	// Unsubscribe from hodor events
	if w.subscriptionID != "" {
		if err := w.contract.Unsubscribe(w.subscriptionID); err != nil {
			zlog.Warn("Failed to unsubscribe from hodor events", 
				zlog.String("key", w.key),
				zlog.Err(err))
		}
	}

	w.state = stateDismissed
	zlog.Info("Dismissed watcher", zlog.String("key", w.key))
	
	return nil
}

// IsActive returns true if watching (active, recovering, or paused)
func (w *watcher[T]) IsActive() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state != stateDismissed
}

// IsRecovering returns true if in recovery mode due to parse error
func (w *watcher[T]) IsRecovering() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state == stateRecovering
}

// Pause temporarily stops callback execution while continuing to watch
func (w *watcher[T]) Pause() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.state == stateDismissed {
		return fmt.Errorf("cannot pause dismissed watcher")
	}

	w.state = statePaused
	zlog.Debug("Paused watcher", zlog.String("key", w.key))
	return nil
}

// Resume restarts callback execution from paused state
func (w *watcher[T]) Resume() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.state == stateDismissed {
		return fmt.Errorf("cannot resume dismissed watcher")
	}

	if w.state == statePaused {
		w.state = stateActive
		zlog.Debug("Resumed watcher", zlog.String("key", w.key))
	}

	return nil
}

// IsPaused returns true if watcher is in paused state
func (w *watcher[T]) IsPaused() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state == statePaused
}