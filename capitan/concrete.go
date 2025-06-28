package capitan

import (
	"zbz/cereal"
	"zbz/zlog"
)

// ByteHandler is the interface that all concrete hooks implement
// Service layer only deals with this interface - no reflection needed
type ByteHandler interface {
	Handle(eventBytes []byte) error
}

// ConcreteInputHook handles typed input events by deserializing bytes
type ConcreteInputHook[T any] struct {
	hookType string
	handler  func(T) error
}

func (h *ConcreteInputHook[T]) Handle(eventBytes []byte) error {
	// Deserialize bytes to our expected type
	var event TypedEvent[T]
	if err := cereal.JSON.Unmarshal(eventBytes, &event); err != nil {
		zlog.Error("Failed to deserialize input event",
			zlog.String("hook_type", h.hookType),
			zlog.Err(err))
		return err
	}

	// Call the actual handler with the typed data
	return h.handler(event.Data)
}

// ConcreteOutputHook handles typed output events by deserializing bytes
type ConcreteOutputHook[T any] struct {
	hookType string
	handler  func(T) error
}

func (h *ConcreteOutputHook[T]) Handle(eventBytes []byte) error {
	// Deserialize bytes to our expected type
	var event TypedEvent[T]
	if err := cereal.JSON.Unmarshal(eventBytes, &event); err != nil {
		zlog.Error("Failed to deserialize output event",
			zlog.String("hook_type", h.hookType),
			zlog.Err(err))
		return err
	}

	// Call the actual handler with the typed data
	return h.handler(event.Data)
}

// ConcreteTransformHook handles typed transform events
type ConcreteTransformHook[TIn, TOut any] struct {
	hookType    string
	outputType  string
	transformer func(TIn) (TOut, error)
}

func (h *ConcreteTransformHook[TIn, TOut]) Handle(eventBytes []byte) error {
	// Deserialize input event
	var inputEvent TypedEvent[TIn]
	if err := cereal.JSON.Unmarshal(eventBytes, &inputEvent); err != nil {
		zlog.Error("Failed to deserialize transform input event",
			zlog.String("hook_type", h.hookType),
			zlog.Err(err))
		return err
	}

	// Transform the data
	outputData, err := h.transformer(inputEvent.Data)
	if err != nil {
		zlog.Error("Transform function failed",
			zlog.String("hook_type", h.hookType),
			zlog.Err(err))
		return err
	}

	// Create output event
	outputEvent := TypedEvent[TOut]{
		Type:      h.outputType,
		Source:    "transform-" + inputEvent.Source,
		Timestamp: inputEvent.Timestamp,
		Data:      outputData,
		Context:   inputEvent.Context,
		Metadata:  inputEvent.Metadata,
	}

	// Serialize output event
	outputBytes, err := cereal.JSON.Marshal(outputEvent)
	if err != nil {
		zlog.Error("Failed to serialize transform output event",
			zlog.String("hook_type", h.hookType),
			zlog.Err(err))
		return err
	}

	// Emit transformed event to service layer
	return serviceManager.emitBytes(h.outputType, outputBytes)
}

// Standard adapter function signatures for consistency
type InputHookFunc[T any] func(T) error
type OutputHookFunc[T any] func(T) error
type TransformHookFunc[TIn, TOut any] func(TIn) (TOut, error)

// Adapter interface for standardized adapter patterns
type Adapter interface {
	Name() string
	Connect() error
	Disconnect() error
}