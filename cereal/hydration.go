package cereal

import (
	"sync"
)

// EventSink interface allows external systems to receive cereal events
// Matches zlog's EventSink pattern for consistency
type EventSink interface {
	EmitCerealEvent(event CerealEvent)
}

// CerealEvent represents events emitted during serialization/validation
type CerealEvent struct {
	Action      string                 `json:"action"`       // "marshal", "unmarshal", "validate", "scope_check"
	ModelType   string                 `json:"model_type"`   
	FieldName   string                 `json:"field_name,omitempty"`
	FieldType   string                 `json:"field_type,omitempty"`
	Permissions []string               `json:"permissions"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CerealService holds the global state and event sink
type CerealService struct {
	eventSink EventSink
	mu        sync.RWMutex
}

// Global cereal service instance
var service = &CerealService{}

// SetEventSink allows external systems (like capitan) to register for events
func SetEventSink(sink EventSink) {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.eventSink = sink
}

// GetEventSink returns the current event sink (useful for testing)
func GetEventSink() EventSink {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return service.eventSink
}

// emitEvent emits an event to the registered sink (if any)
func emitEvent(event CerealEvent) {
	service.mu.RLock()
	sink := service.eventSink
	service.mu.RUnlock()
	
	if sink != nil {
		// Emit asynchronously to avoid blocking serialization
		go sink.EmitCerealEvent(event)
	}
}

// Convenience functions for emitting specific event types

func emitMarshalEvent(modelType string, permissions []string, success bool, err error) {
	event := CerealEvent{
		Action:      "marshal",
		ModelType:   modelType,
		Permissions: permissions,
		Success:     success,
	}
	if err != nil {
		event.Error = err.Error()
	}
	emitEvent(event)
}

func emitUnmarshalEvent(modelType string, permissions []string, success bool, err error) {
	event := CerealEvent{
		Action:      "unmarshal",
		ModelType:   modelType,
		Permissions: permissions,
		Success:     success,
	}
	if err != nil {
		event.Error = err.Error()
	}
	emitEvent(event)
}

func emitFieldScopeEvent(modelType, fieldName, fieldType string, permissions []string, granted bool) {
	event := CerealEvent{
		Action:      "scope_check",
		ModelType:   modelType,
		FieldName:   fieldName,
		FieldType:   fieldType,
		Permissions: permissions,
		Success:     granted,
		Metadata: map[string]interface{}{
			"access_granted": granted,
		},
	}
	emitEvent(event)
}

func emitValidationEvent(modelType, fieldName string, success bool, err error) {
	event := CerealEvent{
		Action:    "validate",
		ModelType: modelType,
		FieldName: fieldName,
		Success:   success,
	}
	if err != nil {
		event.Error = err.Error()
	}
	emitEvent(event)
}

// Auto-hydration detection for capitan integration
// This runs when cereal package is imported and capitan is available

type CapitanEventSink interface {
	EmitCerealEvent(event CerealEvent)
}

// detectCapitan attempts to find capitan service for auto-hydration
func detectCapitan() CapitanEventSink {
	// This will be enhanced when capitan adds cereal hydration
	// For now, return nil - capitan will manually register
	return nil
}

// init performs auto-hydration if capitan is available
func init() {
	if capitanSink := detectCapitan(); capitanSink != nil {
		SetEventSink(capitanSink)
	}
}