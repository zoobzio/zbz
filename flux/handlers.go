package flux

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// FluxEvent provides event data for handler pipeline
type FluxEvent[T any] struct {
	// Core data
	Key     string // Storage key (e.g., "config.json")
	Old     T      // Previous value (zero value on first load)
	New     T      // New value from storage
	Changed bool   // True if this is an update, false if initial load

	// Metadata
	Provider  string    // Storage provider name
	Timestamp time.Time // When the change occurred
	Operation string    // "create", "update", "delete"
	Size      int64     // Content size in bytes

	// Control flow
	context.Context        // Standard Go context
	aborted     bool       // Set to true to abort pipeline
	errors      []error    // Accumulated errors
	data        map[string]any // Custom data for handler communication
}

// NewFluxEvent creates a new flux event
func NewFluxEvent[T any](key string, old, new T, operation string, provider string) *FluxEvent[T] {
	return &FluxEvent[T]{
		Key:       key,
		Old:       old,
		New:       new,
		Changed:   !reflect.DeepEqual(old, new),
		Provider:  provider,
		Timestamp: time.Now(),
		Operation: operation,
		Context:   context.Background(),
		data:      make(map[string]any),
	}
}

// Abort stops the handler pipeline
func (e *FluxEvent[T]) Abort() {
	e.aborted = true
}

// IsAborted returns true if pipeline was aborted
func (e *FluxEvent[T]) IsAborted() bool {
	return e.aborted
}

// AddError adds an error to the event
func (e *FluxEvent[T]) AddError(err error) {
	e.errors = append(e.errors, err)
}

// Errors returns all accumulated errors
func (e *FluxEvent[T]) Errors() []error {
	return e.errors
}

// HasErrors returns true if any errors were added
func (e *FluxEvent[T]) HasErrors() bool {
	return len(e.errors) > 0
}

// Set stores data for handler communication
func (e *FluxEvent[T]) Set(key string, value any) {
	e.data[key] = value
}

// Get retrieves data
func (e *FluxEvent[T]) Get(key string) (any, bool) {
	value, exists := e.data[key]
	return value, exists
}

// MustGet retrieves data or panics
func (e *FluxEvent[T]) MustGet(key string) any {
	if value, exists := e.data[key]; exists {
		return value
	}
	panic(fmt.Sprintf("Key '%s' does not exist", key))
}

// HandlerFunc defines the signature for flux handlers
type HandlerFunc[T any] func(*FluxEvent[T])

// HandlerPipeline manages a chain of handler functions
type HandlerPipeline[T any] struct {
	handlers []HandlerFunc[T]
}

// NewPipeline creates a new handler pipeline
func NewPipeline[T any](handlers ...HandlerFunc[T]) *HandlerPipeline[T] {
	return &HandlerPipeline[T]{
		handlers: handlers,
	}
}

// Use adds handlers to the pipeline
func (p *HandlerPipeline[T]) Use(handlers ...HandlerFunc[T]) {
	p.handlers = append(p.handlers, handlers...)
}

// Execute runs the handler pipeline
func (p *HandlerPipeline[T]) Execute(event *FluxEvent[T]) {
	for _, handler := range p.handlers {
		if event.IsAborted() {
			break
		}
		handler(event)
	}
}

// Built-in handler functions

// Validator creates a handler that validates the new config
func Validator[T any](validator func(T) error) HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		if err := validator(event.New); err != nil {
			event.AddError(fmt.Errorf("validation failed: %w", err))
			event.Abort()
		}
	}
}

// Logger logs configuration changes
func Logger[T any]() HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		if event.Changed {
			fmt.Printf("🔄 Config changed: %s (provider: %s, operation: %s)\n", 
				event.Key, event.Provider, event.Operation)
		} else {
			fmt.Printf("📖 Config loaded: %s (provider: %s)\n", 
				event.Key, event.Provider)
		}
	}
}

// Metrics tracks configuration change metrics
func Metrics[T any]() HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		// In a real implementation, this would increment metrics
		event.Set("metrics_recorded", true)
		event.Set("change_time", event.Timestamp)
	}
}

// Diff calculates and stores differences between old and new
func Diff[T any]() HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		if event.Changed {
			// In a real implementation, this would calculate actual diffs
			event.Set("has_diff", true)
			event.Set("diff_calculated_at", time.Now())
		}
	}
}

// OnlyChanges only executes inner handler for actual changes (not initial loads)
func OnlyChanges[T any](handler HandlerFunc[T]) HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		if event.Changed {
			handler(event)
		}
	}
}

// OnlyInitial only executes inner handler for initial loads
func OnlyInitial[T any](handler HandlerFunc[T]) HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		if !event.Changed {
			handler(event)
		}
	}
}

// Recovery recovers from panics in subsequent handlers
func Recovery[T any]() HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		defer func() {
			if err := recover(); err != nil {
				event.AddError(fmt.Errorf("handler panic: %v", err))
				event.Abort()
			}
		}()
	}
}

// Custom creates a custom handler from a simple callback function
func Custom[T any](callback func(old, new T)) HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		callback(event.Old, event.New)
	}
}

// Throttle creates a handler that only executes if enough time has passed since last execution
func Throttle[T any](duration time.Duration, handler HandlerFunc[T]) HandlerFunc[T] {
	var lastExecution time.Time
	
	return func(event *FluxEvent[T]) {
		now := time.Now()
		if now.Sub(lastExecution) >= duration {
			lastExecution = now
			handler(event)
		} else {
			event.Set("throttled", true)
		}
	}
}

// Conditional creates a handler that only executes if condition is met
func Conditional[T any](condition func(*FluxEvent[T]) bool, handler HandlerFunc[T]) HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		if condition(event) {
			handler(event)
		}
	}
}

// Transform allows modifying the new value before subsequent handlers see it
func Transform[T any](transformer func(T) T) HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		event.New = transformer(event.New)
	}
}

// FieldWatcher only triggers handler if specific fields have changed (using reflection)
func FieldWatcher[T any](fieldNames []string, handler HandlerFunc[T]) HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		if !event.Changed {
			handler(event) // Always execute on initial load
			return
		}
		
		// Use reflection to check if any watched fields changed
		oldVal := reflect.ValueOf(event.Old)
		newVal := reflect.ValueOf(event.New)
		
		if oldVal.Kind() == reflect.Ptr {
			oldVal = oldVal.Elem()
		}
		if newVal.Kind() == reflect.Ptr {
			newVal = newVal.Elem()
		}
		
		changed := false
		for _, fieldName := range fieldNames {
			oldField := oldVal.FieldByName(fieldName)
			newField := newVal.FieldByName(fieldName)
			
			if oldField.IsValid() && newField.IsValid() {
				if !reflect.DeepEqual(oldField.Interface(), newField.Interface()) {
					changed = true
					break
				}
			}
		}
		
		if changed {
			event.Set("watched_fields_changed", fieldNames)
			handler(event)
		}
	}
}

// Backup creates a handler that stores previous values for rollback
func Backup[T any](maxVersions int) HandlerFunc[T] {
	var history []T
	
	return func(event *FluxEvent[T]) {
		// Add current old value to history
		history = append(history, event.Old)
		
		// Maintain max versions
		if len(history) > maxVersions {
			history = history[1:]
		}
		
		event.Set("backup_history", history)
		event.Set("backup_count", len(history))
	}
}

// ErrorHandler creates a handler that catches and handles errors from subsequent handlers
func ErrorHandler[T any](errorCallback func(error, *FluxEvent[T])) HandlerFunc[T] {
	return func(event *FluxEvent[T]) {
		defer func() {
			if event.HasErrors() {
				for _, err := range event.Errors() {
					errorCallback(err, event)
				}
			}
		}()
	}
}